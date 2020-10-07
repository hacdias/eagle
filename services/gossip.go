package services

import (
	"bytes"
	"log"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

func (s *Services) getWebmentionTargets(entry *HugoEntry) ([]string, error) {
	html, err := s.Hugo.GetEntryHTML(entry.ID)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, err
	}

	targets := []string{}

	doc.Find(".h-entry .e-content p").Each(func(i int, q *goquery.Selection) {
		val, ok := q.Attr("href")
		if !ok {
			return
		}

		u, err := url.Parse(val)
		if err != nil {
			// ???
			return
		}

		if u.Host == "" {
			u.Host = s.cfg.Domain
		}

		targets = append(targets, u.String())
	})

	return targets, err
}

// Gossip takes care of the interactions of a certain post with the world.
func (s *Services) Gossip(entry *HugoEntry, syn *Syndication) {
	url := s.cfg.Domain + entry.ID

	targets, err := s.getWebmentionTargets(entry)
	if err != nil {
		log.Printf("failed to get webmentions targets: %s", err)
	} else {
		err = s.Webmentions.Send(url, targets...)
		if err != nil {
			log.Printf("failed to send webmentions: %s", err)
		}
	}

	// TODO: syndicate to twitter
}
