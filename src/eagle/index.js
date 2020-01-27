const { join } = require('path')
const pLimit = require('p-limit')

const micropub = require('./micropub')
const createWebmention = require('./webmentions')
const createPOSSE = require('./posse')
const createXRay = require('./xray')
const HugoService = require('./hugo')
const createTwitter = require('./twitter')
const createGit = require('./git')
const createTelegram = require('./telegram')

function createEagle ({ domain, ...config }) {
  const limit = pLimit(1)

  const hugo = new HugoService({
    ...config.hugo,
    domain
  })

  const xray = createXRay({
    domain,
    twitter: config.twitter,
    entrypoint: config.xrayEntrypoint,
    dir: join(hugo.dataDir, 'xray')
  })

  const git = createGit({
    cwd: hugo.dir
  })

  const webmentions = createWebmention({
    token: config.telegraphToken,
    domain: domain,
    xray,
    dir: join(hugo.dataDir, 'mentions')
  })

  const twitter = createTwitter(config.twitter)

  const telegram = createTelegram({
    ...config.telegram,
    git,
    hugo
  })

  const posse = createPOSSE({
    twitter
  })

  const wrap = async (fn) => {
    try {
      const res = await fn()
      return res
    } catch (e) {
      telegram.sendError(e)
      throw e
    }
  }

  const wrapAndLimit = (fn) => wrap(() => limit(fn))

  const receiveWebmention = (webmention) => wrapAndLimit(async () => {
    await webmentions.receive(webmention)
    git.commit(`webmention from ${webmention.post.url}`)
    hugo.build()
    telegram.send(`💬 Received webmention: ${webmention.target}`)
  })

  const receiveMicropub = (req, res, data) => wrapAndLimit(async () => {
    const { meta, content, slug, type, relatedURL } = micropub.createPost(data)

    if (relatedURL) {
      try {
        await xray.requestAndSave(relatedURL)
      } catch (e) {
        telegram.sendError(e)
      }
    }

    const { post } = await hugo.newEntry({ meta, content, slug })
    const url = `${domain}${post}`

    res.redirect(202, url)

    git.commit(`add ${post}`)
    hugo.build()

    telegram.send(`📄 Post published: ${url}`)

    try {
      const html = await hugo.getEntryHTML(post)
      await webmentions.sendFromContent({ url, body: html })
    } catch (e) {
      telegram.sendError(e)
    }

    try {
      const syndication = await posse.analysePost({
        content,
        url,
        type,
        commands: data.commands,
        relatedURL
      })

      if (syndication.length >= 1) {
        await updateMicropub(null, null, {
          url,
          update: {
            add: {
              syndication
            }
          }
        })
      }
    } catch (e) {
      telegram.sendError(e)
    }

    if (!relatedURL) {
      return
    }

    try {
      await webmentions.send({ source: url, targets: [relatedURL] })
    } catch (e) {
      telegram.sendError(e)
    }
  })

  const updateMicropub = (req, res, data) => wrapAndLimit(async () => {
    const post = data.url.replace(domain, '', 1)
    let entry = await hugo.getEntry(post)
    entry = micropub.updatePost(entry, data)

    if (res) {
      res.redirect(202, data.url)

      // Update updated date!
      if (!entry.meta.publishDate && entry.meta.date) {
        entry.meta.publishDate = entry.meta.date
      }

      entry.meta.date = new Date()
    }

    await hugo.saveEntry(post, entry)
    git.commit(`update ${post}`)
    hugo.build()
    telegram.send(`📄 Post updated: ${data.url}`)
  })

  const sourceMicropub = async (url) => wrap(async () => {
    if (!url.startsWith(domain)) {
      throw new Error('invalid request')
    }

    const post = url.replace(domain, '', 1)
    const { meta, content } = await hugo.getEntry(post)

    return {
      type: ['h-entry'],
      properties: {
        ...meta.properties,
        name: [meta.title],
        content: [content]
      }
    }
  })

  return Object.freeze({
    // services
    telegram,
    hugo,
    xray,
    git,
    webmentions,
    twitter,
    posse,

    receiveWebmention,
    updateMicropub,
    sourceMicropub,
    receiveMicropub
  })
}

createEagle.fromEnvironment = () => createEagle({
  xrayEntrypoint: process.env.XRAY_ENTRYPOINT,
  telegraphToken: process.env.TELEGRAPH_TOKEN,
  domain: process.env.DOMAIN,
  hugo: {
    dir: process.env.HUGO_DIR,
    publicDir: process.env.HUGO_PUBLIC_DIR
  },
  twitter: {
    apiKey: process.env.TWITTER_API_KEY,
    apiSecret: process.env.TWITTER_API_SECRET,
    accessToken: process.env.TWITTER_ACCESS_TOKEN,
    accessTokenSecret: process.env.TWITTER_ACCESS_TOKEN_SECRET
  },
  telegram: {
    token: process.env.TELEGRAM_TOKEN,
    chatID: parseInt(process.env.TELEGRAM_CHAT_ID)
  }
})

module.exports = createEagle
