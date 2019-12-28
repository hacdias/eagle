const got = require('got')

module.exports = async url => {
  const { body } = await got.post(`${process.env.XRAY_ENTRYPOINT}/parse`, {
    form: {
      url,
      twitter_api_key: process.env.TWITTER_API_KEY,
      twitter_api_secret: process.env.TWITTER_API_SECRET,
      twitter_access_token: process.env.TWITTER_ACCESS_TOKEN,
      twitter_access_token_secret: process.env.TWITTER_ACCESS_TOKEN_SECRET
    },
    responseType: 'json'
  })

  return body
}
