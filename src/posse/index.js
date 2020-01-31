const debug = require('debug')('eagle:posse')
const createTwitter = require('./twitter')

module.exports = function createPOSSE (conf) {
  const twitter = createTwitter(conf.twitter)

  const relatesToTwitter = async ({ url, type, status }) => {
    const id = new URL(url).pathname.split('/').pop()
    let res, syndication

    switch (type) {
      case 'like':
        await twitter.like(id)
        break
      case 'repost':
        await twitter.retweet(id)
        break
      case 'reply':
        res = await twitter.tweet({
          status: status,
          inReplyTo: id
        })
        syndication = `https://twitter.com/hacdias/status/${res.id_str}`
        break
      default:
        break
    }

    return syndication
  }

  return async ({ content, url, type, commands, relatedURL }) => {
    const syndications = []
    const errors = []

    const smallContent = content.length <= 280
      ? content
      : `${content.substr(0, 230).trim()}... ${url}`

    if (commands['mp-syndicate-to'] && commands['mp-syndicate-to'].includes('twitter') && !relatedURL) {
      try {
        const res = await twitter.tweet({ status: smallContent })
        const url = `https://twitter.com/hacdias/status/${res.id_str}`
        syndications.push(url)
      } catch (e) {
        debug('could not syndicate to twitter: %s', e.stack)
        errors.push(e)
      }
    }

    if (relatedURL && relatedURL.startsWith('https://twitter.com')) {
      try {
        const syndicate = await relatesToTwitter({
          url: relatedURL,
          type,
          status: smallContent
        })

        if (syndicate) {
          syndications.push(syndicate)
        }
      } catch (e) {
        debug('could not syndicate to twitter: %s', e.stack)
        errors.push(e)
      }
    }

    if (errors.length >= 1) {
      throw new Error(errors.reduce((prev, curr) => prev + `${curr.stack}`, ''))
    }

    return syndications
  }
}
