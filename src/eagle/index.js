const { join } = require('path')
const { parse } = require('node-html-parser')
const fs = require('fs-extra')
const pLimit = require('p-limit')
const crypto = require('crypto')
const debug = require('debug')('eagle')

const webmentions = require('./webmentions')
const { configuredXray } = require('./xray')
const { configuredHugo } = require('./hugo')
const { configuredGit } = require('./git')
const parseMicropub = require('./micropub')

class Eagle {
  constructor ({
    domain,
    hugo,
    twitter,
    telegraphToken,
    xrayEntrypoint
  }) {
    this.limit = pLimit(1)
    this.hugoOpts = hugo
    this.domain = domain
    this.telegraphToken = telegraphToken

    this.hugo = configuredHugo(hugo)

    this.xray = configuredXray({
      twitter,
      entrypoint: xrayEntrypoint
    })

    this.git = configuredGit({
      cwd: hugo.dir
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

  async sendContentWebmentions (url) {
    debug('will scrap %s for webmentions', url)
    const path = this._urlToLocal(url)
    const file = (await fs.readFile(path)).toString()
    const ray = await this.xray({ url, body: file })
    const parsed = parse(ray.data.content.html)
    const targets = parsed.querySelectorAll('a')
      .map(p => p.attributes.href)

    debug('found webmentions: %o', targets)

    await webmentions.send({
      source: url,
      targets,
      token: this.telegraphToken
    })
  }

  async receiveWebMention () {
    /*
     return this.limit(() => {
      const dataPath = path.join(
        this.contentDir,
        webmention.target.replace('https://hacdias.com/', '', 1),
        'data'
      )

      fs.ensureDirSync(dataPath)
      fs.writeFileSync(
        path.join(dataPath, 'index.md'),
        '---\nheadless: true\n---'
      )

      const dataFile = path.join(dataPath, 'webmentions.json')

      if (!fs.existsSync(dataFile)) {
        fs.outputJSONSync(dataFile, [webmention.post], {
          spaces: 2
        })
      } else {
        const arr = fs.readJSONSync(dataFile)
        const inArray = arr.filter(a => a['wm-id'] === webmention.post['wm-id']).length !== 0

        if (!inArray) {
          arr.push(webmention.post)
          fs.outputJSONSync(dataFile, arr, {
            spaces: 2
          })
        }
      }

      // this._gitCommit(`webmention from ${webmention.post.url}`)
      // this._hugoBuild()
      // git.push({ cwd: this.dir })
      */
  }

  async receiveMicropub (data) {
    const {
      meta,
      content,
      slug,
      relatedTo,
      titleWasEmpty
    } = await parseMicropub(data)

    return this.limit(async () => {
      if (relatedTo) {
        const data = await this._xrayAndSave(relatedTo.url)

        if (titleWasEmpty && data.name) {
          meta.title = data.name
        }
      }

      const path = this.hugo.makePost({
        meta,
        content,
        slug
      })

      const url = `${this.domain}${path}`

      this.git.commit(`add ${path}`)
      this.hugo.build()

      // Async actions
      ;(async () => {
        this.sendContentWebmentions(url)
        this.limit(() => {
          this.git.push()
        })

        if (!relatedTo) {
          return
        }

        webmentions.send({
          source: url,
          targets: [relatedTo.url],
          token: this.telegraphToken
        })
      })()

      return url
    })
  }

  _urlToLocal (url) {
    if (!url.startsWith(this.domain)) {
      throw new Error('url must start with domain')
    }

    let uri = url.replace(this.domain, '', 1)
    if (uri.endsWith('/')) {
      uri += 'index.html'
    }

    return join(this.hugoOpts.publicDir, uri)
  }

  async _xrayAndSave (url) {
    debug('gonna xray %s', url)

    try {
      const sha256 = crypto.createHash('sha256').update(url).digest('hex')
      const rxayDir = join(this.hugoOpts.dir, 'data', 'xray')
      const xrayFile = join(rxayDir, `${sha256}.json`)

      if (!await fs.exists(xrayFile)) {
        const data = await this.xray({ url })

        if (data.code !== 200) {
          return
        }

        await fs.outputJSON(xrayFile, data.data, {
          spaces: 2
        })

        debug('%s successfully xrayed', url)
        return data.data
      } else {
        debug('%s already xrayed', url)
        return fs.readJson(xrayFile)
      }
    } catch (e) {
      debug('could not xray %s: %s', url, e.toString())
    }
  }
}

module.exports = Eagle
