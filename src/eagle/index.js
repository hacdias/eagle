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

class Eagle {
  constructor ({
    domain,
    hugo,
    twitter,
    telegraphToken,
    xrayEntrypoint
  }) {
    this.limit = pLimit(1)
    this.domain = domain

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
      }
    })
  }

  async receiveWebMention (webmention, { skipGit, skipBuild } = {}) {
    this.limit(async () => {
      await this.webmentions.receive(webmention)

      if (!skipGit) {
        this.git.commit(`webmention from ${webmention.post.url}`)
      }

      if (!skipBuild) {
        this.hugo.build()
      }

      if (!skipGit) {
        this.git.push()
      }
    })
  }

  async receiveMicropub (data) {
    return this.limit(async () => {
      const noTitle = data.properties.name
        ? data.properties.name.length === 0
        : false

      const { meta, content, slug, type, relatedURL } = Micropub.createPost(data)

      this.location.updateEntry(meta)

      if (relatedURL) {
        const data = await this.xray.requestAndSave(relatedURL)

        if (noTitle && data.name) {
          meta.title = data.name
        }
      }

      const { post } = this.hugo.newEntry({ meta, content, slug })
      const url = `${this.domain}${post}`

      this.git.commit(`add ${post}`)
      this.hugo.build()

      // Async actions
      ;(async () => {
        const html = await this.hugo.getEntryHTML(post)

        await this.webmentions.sendFromContent({ url, body: html })

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
            add: {
              syndication
            }
          })
        }

        if (!relatedURL) {
          return
        }

        this.webmentions.send({
          source: url,
          targets: [relatedURL]
        })
      })()

      return url
    })
  }

  async updateMicropub (data) {
    return this.limit(async () => {
      const post = data.url.replace(this.domain, '', 1)
      let entry = await this.hugo.getEntry(post)
      entry = Micropub.updatePost(entry, data)
      await this.hugo.saveEntry(post, entry)

      this.git.commit(`update ${post}`)
      this.hugo.build()

      return data.url
    })
  }
}

module.exports = Eagle
