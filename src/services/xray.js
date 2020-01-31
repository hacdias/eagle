const got = require('got')
const crypto = require('crypto')
const fs = require('fs-extra')
const { join } = require('path')
const debug = require('debug')('eagle:xray')

function sha256 (data) {
  return crypto.createHash('sha256').update(data).digest('hex')
}

module.exports = function createXRay ({ domain, entrypoint, twitter, dir }) {
  const makeOptions = () => {
    return {
      form: {
        twitter_api_key: twitter.apiKey,
        twitter_api_secret: twitter.apiSecret,
        twitter_access_token: twitter.accessToken,
        twitter_access_token_secret: twitter.accessTokenSecret
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

    const res = await got.post(`${entrypoint}/parse`, options)

    if (res.body.data && res.body.data.published) {
      res.body.data.published = new Date(res.body.data.published).toISOString()
    }

    return res.body
  }

  const requestAndSave = async (url) => {
    debug('gonna xray %s', url)

    try {
      const file = join(dir, `${sha256(url)}.json`)

      if (url.startsWith('/')) {
        url = `${domain}${url}`
      }

      if (!await fs.exists(file)) {
        const data = await request({ url })

        if (data.code !== 200) {
          return
        }

        await fs.outputJSON(file, data.data, {
          spaces: 2
        })

        debug('%s successfully xrayed', url)
        return data.data
      } else {
        debug('%s already xrayed: %s', url, file)
        return fs.readJson(file)
      }
    } catch (e) {
      debug('could not xray %s: %s', url, e.stack)
      throw e
    }
  }

  return Object.freeze({
    request,
    requestAndSave
  })
}
