const got = require('got')
const debug = require('debug')('eagle:webmentions')
const { parse } = require('node-html-parser')
const { sha256 } = require('./utils')
const fs = require('fs-extra')
const { join, extname } = require('path')

const types = Object.freeze({
  'like-of': 'like',
  'repost-of': 'repost',
  'mention-of': 'mention',
  'in-reply-to': 'reply'
})

async function uploadToCdn (entry, cdn) {
  try {
    const ext = extname(entry.author.photo)
    const base = sha256(entry.author.photo)
    const stream = got.stream(entry.author.photo)
    const url = await cdn.upload(stream, `/webmentions/${base}${ext}`)
    entry.author.photo = url
  } catch (e) {
    debug('could not upload photo to cdn %s: %s', entry.author.photo, e.stack)
  }

  return entry
}

module.exports = function createWebmention ({ token, git, domain, dir, cdn }) {
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

  const receive = async (webmention) => {
    const permalink = webmention.target.replace(domain, '', 1)
    const hash = sha256(permalink)
    const file = join(dir, `${hash}.json`)

    if (!await fs.exists(file)) {
      await fs.outputJSON(file, [])
    }

    const mentions = await fs.readJSON(file)

    if (webmention.deleted) {
      const newMentions = mentions.filter(m => m.url !== webmention.source)
      await fs.outputJSON(file, newMentions, { spaces: 2 })
      await git.commit(`deleted webmention from ${webmention.source}`)
      return
    }

    if (mentions.find(m => m['wm-id'] === webmention.post['wm-id'])) {
      debug('duplicated webmention for %s: %s', permalink, webmention.post['wm-id'])
      return
    }

    let entry = {
      type: types[webmention.post['wm-property']] || 'mention',
      url: webmention.post.url || webmention.post['wm-source'],
      date: new Date(webmention.post.published || webmention.post['wm-received']),
      private: webmention.post['wm-private'] || false,
      'wm-id': webmention.post['wm-id'],
      content: webmention.post.content,
      author: webmention.post.author
    }

    delete entry.author.type

    if (webmention.post['swarm-coins']) {
      entry['swarm-coins'] = webmention.post['swarm-coins']
    }

    // upload avatar to cdn
    if (entry.author && entry.author.photo) {
      entry = await uploadToCdn(entry, cdn)
    }

    mentions.push(entry)
    await fs.outputJSON(file, mentions, { spaces: 2 })
    await git.commit(`webmention from ${entry.url}`)
  }

  return Object.freeze({
    send,
    sendFromContent,
    receive
  })
}
