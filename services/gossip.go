package services

import (
	"bytes"

	"github.com/PuerkitoBio/goquery"
)

// Gossip takes care of the interactions of a certain post with the world.
func (s *Services) Gossip(entry *HugoEntry, syn *Syndication) {
	url := s.cfg.Domain + entry.ID

	html, err := s.Hugo.GetEntryHTML(entry.ID)
	if err != nil {
		return
	}

	goquery.NewDocumentFromReader(bytes.NewReader(html))

}

/*

const getMentions = async (url, body) => {
  debug('will scrap %s for webmentions', url)
  const parsed = parse(body)

  const targets = parsed.querySelectorAll('.h-entry .e-content a')
    .map(p => p.attributes.href)
    .map(href => {
      try {
        const u = new URL(href, url)
        return u.href
      } catch (_) {
        return href
      }
    })

  debug('found webmentions: %o', targets)
  return targets
}

const sendWebmentions = async (post, url, related, services) => {
  const { hugo, notify, webmentions } = services
  const targets = [...related]

  try {
    const html = await hugo.getEntryHTML(post)
    const mentions = await getMentions(url, html)
    targets.push(...mentions)
  } catch (err) {
    notify.sendError(err)
  }

  try {
    await webmentions.send({ source: url, targets })
  } catch (err) {
    notify.sendError(err)
  }
}
*/
