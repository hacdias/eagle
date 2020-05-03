const debug = require('debug')('eagle:webmentions')
const got = require('got')
const fs = require('fs-extra')
const { join, extname } = require('path')
const sha256 = require('../utils/sha256')

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

function loadRedirects (file) {
  try {
    return fs.readFileSync(file)
      .toString()
      .split('\n')
      .filter(p => !!p)
      .map(e => e.split(' '))
      .reduce((acc, [oldLink, newLink]) => {
        if (acc[oldLink]) {
          throw new Error('must not exist')
        }

        acc[oldLink] = newLink
        return acc
      }, {})
  } catch (e) {
    debug('cant load redirects %s', e.stack)
  }
}

module.exports = function createWebmention ({ redirectsFile, storeDir, telegraphToken, domain, git, cdn }) {
  let redirects = loadRedirects(redirectsFile) || {}

  const send = async ({ source, targets }) => {
    for (const target of targets) {
      const webmention = { source, target }

      debug('outgoing webmention %o', webmention)

      const { statusCode, body } = await got.post('https://telegraph.p3k.io/webmention', {
        form: {
          ...webmention,
          token: telegraphToken
        },
        responseType: 'json',
        throwHttpErrors: false
      })

      if (statusCode >= 400) {
        debug('outgoing webmention failed: %o', body)
      } else {
        debug('outgoing webmention succeeded', webmention)
      }
    }
  }

  const receive = async (webmention, ignoreGit = false) => {
    redirects = loadRedirects(redirectsFile) || redirects

    let permalink = webmention.target.replace(domain, '', 1)

    if (redirects[permalink]) {
      permalink = redirects[permalink]
    }

    const hash = sha256(permalink)
    const file = join(storeDir, `${hash}.json`)

    const mentions = await fs.exists(file)
      ? await fs.readJSON(file)
      : []

    if (webmention.deleted) {
      const newMentions = mentions.filter(m => m.url !== webmention.source)
      await fs.outputJSON(file, newMentions, { spaces: 2 })
      if (!ignoreGit) await git.commit(`deleted webmention from ${webmention.source}`)
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

    // upload avatar to cdn
    if (entry.author && entry.author.photo) {
      entry = await uploadToCdn(entry, cdn)
    }

    debug('saving received webmention from %', entry.url)
    mentions.push(entry)
    await fs.outputJSON(file, mentions, { spaces: 2 })
    if (!ignoreGit) await git.commit(`webmention from ${entry.url}`)
  }

  return Object.freeze({
    send,
    receive
  })
}
