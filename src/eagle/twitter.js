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

  got () {
    got(arguments, {
      headers: oauth.toHeader(oauth.authorize({ url, method: 'GET' }, token)),
      responseType: 'json'
    })
  }

  timeline () {
    const url = 'https://api.twitter.com/1.1/statuses/home_timeline.json'

    got(url, {
      headers: this.oauth.toHeader(this.oauth.authorize({ url, method: 'GET' }, this.token)),
      responseType: 'json'
    })
  }
}
