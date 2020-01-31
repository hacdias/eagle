const OAuth = require('oauth-1.0a')
const got = require('got')
const crypto = require('crypto')
const debug = require('debug')('eagle:twitter')

module.exports = function createTwitter ({ apiKey, apiSecret, accessToken, accessTokenSecret }) {
  const oauth = OAuth({
    consumer: {
      key: apiKey,
      secret: apiSecret
    },
    signature_method: 'HMAC-SHA1',
    hash_function: (baseString, key) => crypto.createHmac('sha1', key).update(baseString).digest('base64')
  })

  const token = {
    key: accessToken,
    secret: accessTokenSecret
  }

  const makeHeaders = (url, method) => {
    return oauth.toHeader(oauth.authorize({ url, method }, token))
  }

  const get = async (url) => {
    const { body } = await got(url, {
      headers: makeHeaders(url, 'GET'),
      responseType: 'json'
    })

    return body
  }

  const post = async (url) => {
    const { body } = await got.post(url, {
      headers: makeHeaders(url, 'POST'),
      responseType: 'json'
    })

    return body
  }

  const timeline = () => {
    return get('https://api.twitter.com/1.1/statuses/home_timeline.json')
  }

  const like = (id) => {
    debug('will like %s', id)
    return post(`https://api.twitter.com/1.1/favorites/create.json?id=${id}`)
  }

  const retweet = (id) => {
    debug('will retweet %s', id)
    return post(`https://api.twitter.com/1.1/statuses/retweet/${id}.json`)
  }

  const tweet = ({ status, inReplyTo }) => {
    debug('tweeting "%s", replying to %s', status, inReplyTo)

    let url = `https://api.twitter.com/1.1/statuses/update.json?status=${rfc3986Encode(status)}`

    if (inReplyTo) {
      url += `&in_reply_to_status_id=${inReplyTo}&auto_populate_reply_metadata=true`
    }

    return post(url)
  }

  return Object.freeze({
    timeline,
    like,
    retweet,
    tweet
  })
}

// RFC 3986
function rfc3986Encode (str) {
  return encodeURIComponent(str).replace(/[!'()*]/g, function (c) {
    return '%' + c.charCodeAt(0).toString(16)
  })
}
