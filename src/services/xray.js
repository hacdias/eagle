const debug = require('debug')('eagle:services:xray')
const got = require('got')
const fs = require('fs-extra')
const { join } = require('path')
const sha256 = require('../utils/sha256')

module.exports = function createXRay ({ apiEndpoint, storeDir, twitterConf, defaultDomain }) {
  const makeOptions = () => {
    return {
      form: {
        twitter_api_key: twitterConf.apiKey,
        twitter_api_secret: twitterConf.apiSecret,
        twitter_access_token: twitterConf.accessToken,
        twitter_access_token_secret: twitterConf.accessTokenSecret
      },
      responseType: 'json'
    }
  }

  const request = async ({ url, body }) => {
    const options = makeOptions()

    if (url) {
      options.form.url = url
    }

    if (body) {
      options.form.body = body
    }

    const res = await got.post(`${apiEndpoint}/parse`, options)

    if (res.body.data && res.body.data.published) {
      res.body.data.published = new Date(res.body.data.published).toISOString()
    }

    return res.body
  }

  const requestAndSave = async (url) => {
    debug('gonna x-ray %s', url)

    const file = join(storeDir, `${sha256(url)}.json`)

    if (url.startsWith('/')) {
      url = `${defaultDomain}${url}`
    }

    if (await fs.exists(file)) {
      debug('%s already x-rayed: %s', url, file)
      return fs.readJson(file)
    }

    const data = await request({ url })

    if (data.code !== 200) {
      return
    }

    await fs.outputJSON(file, data.data, {
      spaces: 2
    })

    debug('%s successfully x-rayed', url)
    return data.data
  }

  return Object.freeze({
    request,
    requestAndSave
  })
}
