const got = require('got')

module.exports = async ({ url, body }, { entrypoint, twitter }) => {
  const options = {
    form: {
      twitter_api_key: twitter.apiKey,
      twitter_api_secret: twitter.apiSecret,
      twitter_access_token: twitter.accessToken,
      twitter_access_token_secret: twitter.accessTokenSecret
    },
    responseType: 'json'
  }

  if (url) {
    options.form.url = url
  }

  if (body) {
    options.form.body = body
  }

  const res = await got.post(`${entrypoint}/parse`, options)
  return res.body
}
