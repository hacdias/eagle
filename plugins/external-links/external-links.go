package externallinks

import (
	"encoding/json"
	"net/http"

	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/server"
)

var (
	_ server.HandlerPlugin = &ExternalLinks{}
)

func init() {
	server.RegisterPlugin("external-links", NewExternalLinks)
}

type ExternalLinks struct{}

func NewExternalLinks(_ *core.Core, _ struct{}) (server.Plugin, error) {
	return &ExternalLinks{}, nil
}

func (el *ExternalLinks) HandlerRoute() string {
	return wellKnownLinksPath
}

func (el *ExternalLinks) Handler(w http.ResponseWriter, r *http.Request, utils *server.PluginWebUtilities) {
	data, err := utils.ReadStaticFile(wellKnownLinksPath)
	if err != nil {
		utils.ErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	var links []linkCollection
	err = json.Unmarshal(data, &links)
	if err != nil {
		utils.ErrorHTML(w, r, http.StatusInternalServerError, err)
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		utils.JSON(w, http.StatusOK, links)
		return
	}

	for _, l := range links {
		if l.Domain == domain {
			utils.JSON(w, http.StatusOK, l)
			return
		}
	}

	utils.ErrorHTML(w, r, http.StatusNotFound, nil)
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

const wellKnownLinksPath = "/.well-known/links"
