const debug = require('debug')('eagle:server:micropub')
const { parse } = require('node-html-parser')

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

module.exports = {
  getMentions
}
