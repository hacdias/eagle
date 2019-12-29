const got = require('got')
const debug = require('debug')('eagle:webmentions')

const send = async ({ token, source, targets }) => {
  for (const target of targets) {
    const webmention = { source, target }

    try {
      debug('outgoing webmention %o', webmention)

      const { statusCode, body } = await got.post('https://telegraph.p3k.io/webmention', {
        form: {
          ...webmention,
          token
        },
        responseType: 'json',
        throwHttpErrors: false
      })

      if (statusCode >= 400) {
        debug('outgoing webmention failed: %o', body)
      } else {
        debug('outgoing webmention succeeded', webmention)
      }
    } catch (e) {
      debug('outgoing webmention failed: %s', e.toString())
    }
  }
}

const receive = () => {

}

module.exports = {
  send,
  receive
}
