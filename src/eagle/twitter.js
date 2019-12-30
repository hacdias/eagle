const OAuth = require('oauth-1.0a')
const got = require('got')
const crypto = require('crypto')

module.exports = class Twitter {
  constructor (opts) {
    this.oauth = OAuth({
      consumer: {
        key: opts.apiKey,
        secret: opts.apiSecret
      },
      signature_method: 'HMAC-SHA1',
      hash_function: (baseString, key) => crypto.createHmac('sha1', key).update(baseString).digest('base64')
    })

    this.token = {
      key: opts.accessToken,
      secret: opts.accessTokenSecret
    }
  }

  _makeHeaders (url, method) {
    return this.oauth.toHeader(this.oauth.authorize({ url, method }, this.token))
  }

  async _get (url) {
    const { body } = await got(url, {
      headers: this._makeHeaders(url, 'GET'),
      responseType: 'json'
    })

    return body
  }

  async _post (url) {
    const { body } = got.post(url, {
      headers: this._makeHeaders(url, 'POST'),
      responseType: 'json'
    })

    return body
  }

  timeline () {
    return this._get('https://api.twitter.com/1.1/statuses/home_timeline.json')
  }

  like (id) {
    return this._post(`https://api.twitter.com/1.1/favorites/create.json?id=${id}`)
  }

  retweet (id) {
    return this._post(`https://api.twitter.com/1.1/statuses/retweet/${id}.json`)
  }
}
