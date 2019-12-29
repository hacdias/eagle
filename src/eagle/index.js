const { join } = require('path')
const { parse } = require('node-html-parser')
const fs = require('fs-extra')
const xray = require('./xray')
const webmentions = require('./webmentions')

class Eagle {
  constructor ({ domain, hugo, twitter, telegraphToken, xrayEntrypoint }) {
    this.domain = domain
    this.hugo = hugo
    this.twitter = twitter
    this.xrayEntrypoint = xrayEntrypoint
    this.telegraphToken = telegraphToken
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

  async sendWebMentions (url) {
    const path = this._urlToLocal(url)
    const file = (await fs.readFile(path)).toString()
    const ray = await this._xray({ url, body: file })
    const parsed = parse(ray.data.content.html)
    const targets = parsed.querySelectorAll('a')
      .map(p => p.attributes.href)

    await webmentions.send({
      source: url,
      targets,
      token: this.telegraphToken
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

    return join(this.hugo.publicDir, uri)
  }

  async _xray (arg) {
    return xray(arg, {
      twitter: this.twitter,
      entrypoint: this.xrayEntrypoint
    })
  }
}

module.exports = Eagle
