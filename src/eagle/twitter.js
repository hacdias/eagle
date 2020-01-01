const OAuth = require('oauth-1.0a')
const got = require('got')
const crypto = require('crypto')
const debug = require('debug')('twitter')

module.exports = class TwitterService {
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
    const { body } = await got.post(url, {
      headers: this._makeHeaders(url, 'POST'),
      responseType: 'json'
    })

    return body
  }

  timeline () {
    return this._get('https://api.twitter.com/1.1/statuses/home_timeline.json')
  }

  like (id) {
    debug('will like %s', id)
    return this._post(`https://api.twitter.com/1.1/favorites/create.json?id=${id}`)
  }

  retweet (id) {
    debug('will retweet %s', id)
    return this._post(`https://api.twitter.com/1.1/statuses/retweet/${id}.json`)
  }

  tweet ({ status, inReplyTo }) {
    debug('tweeting "%s", replying to %s', status, inReplyTo)

    let url = `https://api.twitter.com/1.1/statuses/update.json?status=${encodeURIComponent(status)}`

    if (inReplyTo) {
      url += `${url}&in_reply_to_status_id=${encodeURIComponent(inReplyTo)}&auto_populate_reply_metadata=true`
    }

    return this._post(url)
  }
}
