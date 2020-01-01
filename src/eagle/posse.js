const debug = require('debug')('eagle:posse')

module.exports = class PosseService {
  constructor ({ twitter }) {
    this.twitter = twitter
  }

  async analysePost ({ content, url, type, commands, relatedURL }) {
    const syndications = []
    const errors = []

    const smallContent = content.length <= 280
      ? content
      : `${content.substr(0, 230).trim()}... ${url}`

    if (commands['mp-syndicate-to'] && commands['mp-syndicate-to'].includes('twitter') && !relatedURL) {
      try {
        const res = await this.twitter.tweet({ status: smallContent })
        const url = `https://twitter.com/hacdias/status/${res.id_str}`
        syndications.push(url)
      } catch (e) {
        debug('could not syndicate to twitter: %s', e.stack)
        errors.push(e)
      }
    }

    if (relatedURL && relatedURL.startsWith('https://twitter.com')) {
      try {
        const syndicate = await this._relatesToTwitter({
          url: relatedURL,
          type,
          status: smallContent
        })

        syndications.push(syndicate)
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

  async _relatesToTwitter ({ url, type, status }) {
    const id = url.split('/').pop()
    let res, syndication

    switch (type) {
      case 'like':
        await this.twitter.like(id)
        break
      case 'repost':
        await this.twitter.retweet(id)
        break
      case 'reply':
        res = await this.twitter.tweet({
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
}
