const { join } = require('path')
const pLimit = require('p-limit')

const Micropub = require('./micropub')

const WebmentionsService = require('./webmentions')
const PosseService = require('./posse')
const XRayService = require('./xray')
const HugoService = require('./hugo')
const TwitterService = require('./twitter')
const LocationService = require('./location')
const GitService = require('./git')
const TelegramService = require('./telegram')

class Eagle {
  constructor ({
    domain,
    hugo,
    twitter,
    telegraphToken,
    xrayEntrypoint,
    telegram
  }) {
    this.limit = pLimit(1)
    this.domain = domain

    this.telegram = new TelegramService(telegram)

    this.hugo = new HugoService({
      ...hugo,
      domain
    })

    this.xray = new XRayService({
      twitter,
      entrypoint: xrayEntrypoint,
      dir: join(this.hugo.dataDir, 'xray')
    })

    this.git = new GitService({
      cwd: hugo.dir
    })

    this.location = new LocationService()

    this.webmentions = new WebmentionsService({
      token: telegraphToken,
      domain: domain,
      xray: this.xray,
      dir: join(this.hugo.dataDir, 'webmentions')
    })

    this.posse = new PosseService({
      twitter: new TwitterService(twitter)
    })
  }

  static fromEnvironment () {
    return new Eagle({
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
        chatID: process.env.TELEGRAM_CHAT_ID
      }
    })
  }

  _wrap (fn) {
    try {
      return fn()
    } catch (e) {
      this.telegram.sendError(e)
      throw e
    }
  }

  _wrapAndLimit (fn) {
    return this._wrap(() => this.limit(fn))
  }

  async receiveWebMention (webmention, { skipGit, skipBuild } = {}) {
    return this._wrapAndLimit(async () => {
      await this.webmentions.receive(webmention)

      if (!skipGit) {
        this.git.commit(`webmention from ${webmention.post.url}`)
      }

      if (!skipBuild) {
        this.hugo.build()
      }
    })
  }

  async receiveMicropub (data) {
    return this._wrapAndLimit(async () => {
      const noTitle = data.properties.name
        ? data.properties.name.length === 0
        : false

      const { meta, content, slug, type, relatedURL } = Micropub.createPost(data)

      try {
        this.location.updateEntry(meta)
      } catch (e) {
        this.telegram.sendError(e)
      }

      if (relatedURL) {
        try {
          const data = await this.xray.requestAndSave(relatedURL)
          if (noTitle && data.name) {
            meta.title = data.name
          }
        } catch (e) {
          this.telegram.sendError(e)
        }
      }

      const { post } = await this.hugo.newEntry({ meta, content, slug })
      const url = `${this.domain}${post}`

      this.git.commit(`add ${post}`)
      this.hugo.build()

      // Async actions
      ;(async () => {
        try {
          const html = await this.hugo.getEntryHTML(post)
          await this.webmentions.sendFromContent({ url, body: html })
        } catch (e) {
          this.telegram.sendError(e)
        }

        try {
          const syndication = await this.posse.analysePost({
            content,
            url,
            type,
            commands: data.commands,
            relatedURL
          })

          if (syndication.length >= 1) {
            await this.updateMicropub({
              url,
              update: {
                add: {
                  syndication
                }
              }
            })
          }
        } catch (e) {
          this.telegram.sendError(e)
        }

        if (!relatedURL) {
          return
        }

        try {
          this.webmentions.send({
            source: url,
            targets: [relatedURL]
          })
        } catch (e) {
          this.telegram.sendError(e)
        }
      })()

      return url
    })
  }

  async updateMicropub (data) {
    return this._wrapAndLimit(async () => {
      const post = data.url.replace(this.domain, '', 1)
      let entry = await this.hugo.getEntry(post)
      entry = Micropub.updatePost(entry, data)
      await this.hugo.saveEntry(post, entry)
      this.git.commit(`update ${post}`)
      this.hugo.build()
      return data.url
    })
  }

  async sourceMicropub (url) {
    return this._wrap(async () => {
      if (!url.startsWith(this.domain)) {
        throw new Error('invalid request')
      }

      const post = url.replace(this.domain, '', 1)
      const { meta, content } = await this.hugo.getEntry(post)

      return {
        type: ['h-entry'],
        properties: {
          ...meta.properties,
          name: [meta.title],
          content: [content]
        }
      }
    })
  }
}

module.exports = Eagle
