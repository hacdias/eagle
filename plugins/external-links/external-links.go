package linkding

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/karlseguin/typed"
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
	"golang.org/x/net/publicsuffix"
)

func init() {
	server.RegisterPlugin("external-links", NewExternalLinks)
}

type ExternalLinks struct {
	fs             *core.FS
	filename       string
	ignoredDomains []string

	links    linkCollections
	linksMap map[string]linkCollection
}

func NewExternalLinks(fs *core.FS, config map[string]interface{}) (server.Plugin, error) {
	filename := typed.New(config).String("filename")
	if filename == "" {
		return nil, errors.New("external-links filename missing")
	}

	el := &ExternalLinks{
		fs:             fs,
		filename:       filename,
		ignoredDomains: typed.New(config).Strings("ignored"),
	}

	fmt.Println(el.ignoredDomains)
	links, err := el.loadDiskLinks()
	if err != nil {
		return nil, err
	}
	el.links = links
	el.linksMap = links.byDomain()

	return el, nil
}

func (el *ExternalLinks) GetAction() (string, func() error) {
	return "Update External Links", el.UpdateExternalLinks
}

func (el *ExternalLinks) GetDailyCron() func() error {
	return el.UpdateExternalLinks
}

func (el *ExternalLinks) GetWebHandler(utils *server.PluginWebUtilities) (string, http.HandlerFunc) {
	return wellKnownLinksPath, func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		if domain == "" {
			utils.JSON(w, http.StatusOK, el.links)
		} else if v, ok := el.linksMap[domain]; ok {
			utils.JSON(w, http.StatusOK, v)
		} else {
			utils.ErrorHTML(w, r, http.StatusNotFound, nil)
		}
	}
}

type link struct {
	SourceURL string `json:"sourceUrl"`
	TargetURL string `json:"targetUrl"`
}

type linkCollection struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
	Links  []link `json:"links"`
}

type linkCollections []linkCollection

func (lc linkCollections) byDomain() map[string]linkCollection {
	linksMap := map[string]linkCollection{}
	for _, l := range lc {
		linksMap[l.Domain] = l
	}
	return linksMap
}

func (e *ExternalLinks) loadDiskLinks() (linkCollections, error) {
	var links []linkCollection
	err := e.fs.ReadJSON(e.filename, &links)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return links, nil
}

func (el *ExternalLinks) UpdateExternalLinks() error {
	ee, err := el.fs.GetEntries(false)
	if err != nil {
		return err
	}

	linksMap := map[string][]link{}
	for _, e := range ee {
		urls, err := core.GetMarkdownURLs(e)
		if err != nil {
			return err
		}

		for _, urlStr := range urls {
			if strings.HasPrefix(urlStr, "/") {
				continue
			}

			u, err := url.Parse(urlStr)
			if err != nil {
				return err
			}

			hostname := u.Hostname()
			if hostname == "" {
				continue
			}

			hostname, err = publicsuffix.EffectiveTLDPlusOne(hostname)
			if err != nil {
				return err
			}

			var ignore bool
			for _, ignoredDomain := range el.ignoredDomains {
				if strings.HasSuffix(ignoredDomain, hostname) {
					ignore = true
					break
				}
			}
			if ignore {
				continue
			}

			if _, ok := linksMap[hostname]; !ok {
				linksMap[hostname] = []link{}
			}

			linksMap[hostname] = append(linksMap[hostname], link{
				SourceURL: e.Permalink,
				TargetURL: u.String(),
			})
		}
	}

	newLinks := linkCollections{}
	for domain, domainLinks := range linksMap {
		sort.SliceStable(domainLinks, func(i, j int) bool {
			return domainLinks[i].SourceURL < domainLinks[j].SourceURL
		})

		newLinks = append(newLinks, linkCollection{
			Domain: domain,
			Count:  len(domainLinks),
			Links:  domainLinks,
		})
	}

	sort.SliceStable(newLinks, func(i, j int) bool {
		if newLinks[i].Count == newLinks[j].Count {
			return newLinks[i].Domain < newLinks[j].Domain
		}

		return newLinks[i].Count > newLinks[j].Count
	})

	oldLinks, err := el.loadDiskLinks()
	if err != nil {
		return err
	}

	if reflect.DeepEqual(oldLinks, newLinks) {
		return nil
	}

	err = el.fs.WriteJSON(el.filename, newLinks, "meta: update external links file")
	if err != nil {
		return err
	}

	el.links = newLinks
	el.linksMap = newLinks.byDomain()
	return err
}

const wellKnownLinksPath = "/.well-known/links"
