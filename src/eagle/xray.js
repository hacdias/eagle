const got = require('got')
const fs = require('fs-extra')
const { join } = require('path')
const debug = require('debug')('eagle:xray')
const sha256 = require('./utils')

module.exports = class XRayService {
  constructor ({ entrypoint, twitter, dir }) {
    this.twitter = twitter
    this.dir = dir
    this.entrypoint = entrypoint
  }

  _makeOptions () {
    return {
      form: {
        twitter_api_key: this.twitter.apiKey,
        twitter_api_secret: this.twitter.apiSecret,
        twitter_access_token: this.twitter.accessToken,
        twitter_access_token_secret: this.twitter.accessTokenSecret
      },
      responseType: 'json'
    }
  }

  async request ({ url, body }) {
    const options = this._makeOptions()

    if (url) {
      options.form.url = url
    }

    if (body) {
      options.form.body = body
    }

    const res = await got.post(`${this.entrypoint}/parse`, options)
    return res.body
  }

  async requestAndSave (url) {
    debug('gonna xray %s', url)

    try {
      const file = join(this.dir, `${sha256(url)}.json`)

      if (url.startsWith('/')) {
        url = `${this.domain}${url}`
      }

      if (!await fs.exists(file)) {
        const data = await this.request({ url })

        if (data.code !== 200) {
          return
        }

        await fs.outputJSON(file, data.data, {
          spaces: 2
        })

        debug('%s successfully xrayed', url)
        return data.data
      } else {
        debug('%s already xrayed', url)
        return fs.readJson(file)
      }
    } catch (e) {
      debug('could not xray %s: %s', url, e.toString())
    }
  }
}
