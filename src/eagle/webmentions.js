const got = require('got')
const { join } = require('path')
const debug = require('debug')('eagle:webmentions')
const { parse } = require('node-html-parser')
const { sha256 } = require('./utils')
const fs = require('fs-extra')

module.exports = function createWebmention ({ token, domain, dir, xray }) {
  const indexPath = join(dir, 'index.json')

  if (!fs.existsSync(indexPath)) {
    fs.outputJSONSync(indexPath, {})
  }

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

  const receive = async (webmention) => {
    const permalink = webmention.target.replace(domain, '', 1)
    const hash = sha256(webmention.post.url)
    const index = await fs.readJSON(indexPath)

    if (!index[permalink]) {
      index[permalink] = {
        likes: [],
        others: []
      }
    }

    const dataFile = join(dir, `${hash}.json`)

    if (!await fs.exists(dataFile)) {
      await fs.outputJson(dataFile, webmention.post, {
        spaces: 2
      })
    }

    if (webmention.post['wm-property'] === 'like-of') {
      if (index[permalink].likes.indexOf(hash) === -1) {
        index[permalink].likes.push(hash)
      }
    } else {
      if (index[permalink].others.indexOf(hash) === -1) {
        index[permalink].others.push(hash)
      }
    }

    await fs.outputJSON(index, index, {
      spaces: 2
    })
  }

  return Object.freeze({
    send,
    sendFromContent,
    receive
  })
}
