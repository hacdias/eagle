const got = require('got')
const debug = require('debug')('eagle:webmentions')
const { parse } = require('node-html-parser')

module.exports = function createWebmention (token) {
  const send = async ({ source, targets }) => {
    for (const target of targets) {
      const webmention = { source, target }

      try {
        debug('outgoing webmention %o', webmention)

        const { statusCode, body } = await got.post('https://telegraph.p3k.io/webmention', {
          form: {
            ...webmention,
            token
          },
          responseType: 'json',
          throwHttpErrors: false
        })

        if (statusCode >= 400) {
          debug('outgoing webmention failed: %o', body)
        } else {
          debug('outgoing webmention succeeded', webmention)
        }
      } catch (e) {
        debug('outgoing webmention failed: %s', e.stack)
        throw e
      }
    }
  }

  const sendFromContent = async ({ url, body }) => {
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

    await send({
      source: url,
      targets
    })
  }

  return Object.freeze({
    send,
    sendFromContent
  })
}
