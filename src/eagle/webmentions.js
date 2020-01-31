const got = require('got')
const debug = require('debug')('eagle:webmentions')
const { parse } = require('node-html-parser')

module.exports = function createWebmention ({ token, domain, dir, xray }) {
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
    const ray = await xray.request({ url, body })

    const targets = []
    const toCheck = ['like-of', 'in-reply-to', 'repost-of']

    for (const param of toCheck) {
      if (Array.isArray(ray.data[param])) {
        targets.push(...ray.data[param])
      }
    }

    if (ray.data.content && ray.data.content.html) {
      const parsed = parse(ray.data.content.html)
      targets.push(
        ...parsed.querySelectorAll('a')
          .map(p => p.attributes.href)
      )
    }

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
