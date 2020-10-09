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
  const ext = extname(entry.author.photo)
  const base = sha256(entry.author.photo)

  try {
    const stream = got.stream(entry.author.photo)
    const url = await cdn.upload(stream, `/webmentions/${base}${ext}`)
    return url
  } catch (e) {
    // who cares?
    debug('could not upload photo to cdn %s: %s', entry.author.photo)
  }

  return ''
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

module.exports = function createWebmention ({ telegraphToken, domain, git, cdn, hugo }) {
  const redirectsFile = join(hugo.publicDir, 'redirects.txt')
  const orphansFile = join(hugo.dataDir, 'mentions', 'orphans.json')
  const privateFile = join(hugo.dataDir, 'mentions', 'private.json')

  let redirects =  {}

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

  const getPermalink = (webmention) => {
    redirects = loadRedirects(redirectsFile) || redirects

    let permalink = webmention.target.replace(domain, '', 1)

    if (redirects[permalink]) {
      permalink = redirects[permalink]
    }

    return permalink
  }

  const receive = async (webmention, ignoreGit = false) => {
    const permalink = getPermalink(webmention)
    const isOrphan = !fs.existsSync(join(hugo.contentDir, permalink))
    const isPrivate = webmention.post ? !!webmention.post['wm-private'] : false
    const storeFile = isOrphan
      ? orphansFile
      : isPrivate
        ? privateFile
        : join(hugo.contentDir, permalink, 'mentions.json')

    let mentions = fs.existsSync(storeFile)
      ? fs.readJSONSync(storeFile)
      : []

    if (webmention.deleted) {
      mentions = mentions.filter(m => m.url !== webmention.source)
      fs.outputJSONSync(storeFile, mentions, { spaces: 2 })
      if (!ignoreGit) return git.commit(`deleted webmention from ${webmention.source}`)
      return
    }

    if (mentions.find(m => m['wm-id'] === webmention.post['wm-id'])) {
      debug('duplicated webmention for %s: %s', permalink, webmention.post['wm-id'])
    }

    const entry = {
      type: types[webmention.post['wm-property']] || 'mention',
      url: webmention.post.url || webmention.post['wm-source'],
      date: new Date(webmention.post.published || webmention.post['wm-received']),
      'wm-id': webmention.post['wm-id'],
      content: webmention.post.content,
      author: webmention.post.author
    }

    delete entry.author.type

    if (entry.author) {
      entry.author.photo = await uploadToCdn(entry, cdn)
    }

    debug('saving received webmention from %s', entry.url)
    mentions.push(entry)
    fs.outputJSONSync(storeFile, mentions, { spaces: 2 })
    if (!ignoreGit) return git.commit(`webmention from ${entry.url}`)
  }

  return Object.freeze({
    send,
    receive,
    _loadRedirects: loadRedirects
  })
}
