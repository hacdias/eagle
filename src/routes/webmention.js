const debug = require('debug')('eagle:routes:webmention')
const { join } = require('path')
const fs = require('fs-extra')
const crypto = require('crypto')
const { ar } = require('./utils')

function sha256 (data) {
  return crypto.createHash('sha256').update(data).digest('hex')
}

module.exports = ({ dir, domain, git, hugo, telegram, queue, secret }) => ar(async (req, res) => {
  debug('incoming webmention')

  if (req.body.secret !== secret) {
    debug('invalid secret')
    return res.sendStatus(403)
  }

  delete req.body.secret

  await queue.add(async () => {
    const webmention = req.body
    const permalink = webmention.target.replace(domain, '', 1)
    const hash = sha256(permalink)
    const file = join(dir, `${hash}.json`)

    if (!await fs.exists(file)) {
      await fs.outputJSON(file, [])
    }

    const mentions = await fs.readJSON(file)

    if (mentions.find(m => m['wm-id'] === webmention.post['wm-id'])) {
      return
    }

    const types = {
      'like-of': 'like',
      'repost-of': 'repost',
      'mention-of': 'mention',
      'in-reply-to': 'reply'
    }

    const entry = {
      type: types[webmention.post['wm-property']] || 'mention',
      url: webmention.post.url || webmention.post['wm-source'],
      date: new Date(webmention.post.published || webmention.post['wm-received']),
      private: webmention.post['wm-private'] || false,
      'wm-id': webmention.post['wm-id'],
      content: webmention.post.content,
      author: webmention.post.author
    }

    delete entry.author.type

    mentions.push(entry)

    await fs.outputJSON(file, mentions, {
      spaces: 2
    })

    git.commit(`webmention from ${req.body.post.url}`)
    res.sendStatus(200)

    try {
      hugo.build()
      telegram.send(`💬 Received webmention: ${req.body.target}`)
    } catch (e) {
      // TODO:
      debug('error on post-webmention processor %s', e.stack)
    }
  })

  debug('webmention handled')
})
