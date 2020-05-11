const debug = require('debug')('eagle:syndicate')

function smallStatus (content, url) {
  return content.length <= 280
    ? content
    : `${content.substr(0, 230).trim()}... ${url}`
}

async function sendToTwitter ({ url, type, postData, postUrl, twitter }) {
  let res

  const opts = {
    status: smallStatus(postData.content, postUrl)
  }

  if (type === 'notes') {
    res = await twitter.tweet(opts)
  } else if (type === 'replies') {
    if (postData.modifiers.includes('+RT')) {
      opts.attachment = url
    } else {
      opts.inReplyTo = new URL(url).pathname.split('/').pop()
    }

    res = await twitter.tweet(opts)
  } else {
    throw new Error('invalid type for twitter syndication' + type)
  }

  return `https://twitter.com/hacdias/status/${res.id_str}`
}

const isTwitterURL = url => url.startsWith('https://twitter.com')

module.exports = async function syndicate (services, postUri, postUrl, postData, commands) {
  commands = commands || {}
  commands['mp-syndicate-to'] = commands['mp-syndicate-to'] || []

  const { twitter, hugo, git, notify } = services
  const { type } = postData

  const syndications = (await Promise.all([
    ...commands['mp-syndicate-to'].map(async service => {
      try {
        if (service === 'twitter') {
          return sendToTwitter({ type: 'notes', postData, postUrl, twitter })
        }
      } catch (err) {
        debug('syndication failed to service %s: %s', service, err.stack)
        notify.sendError(err)
      }

      debug('syndication to %s does not exist', service)
    }),
    ...postData.related.map(async url => {
      try {
        if (isTwitterURL(url)) {
          return sendToTwitter({ url, type, postData, postUrl, twitter })
        }
      } catch (err) {
        debug('syndication failed to service %s: %s', url, err.stack)
        notify.sendError(err)
      }

      debug('syndication to %s unknown', url)
    })
  ])).filter(url => !!url)

  if (syndications.length === 0) {
    return
  }

  try {
    const { meta, content } = await hugo.getEntry(postUri)
    meta.properties = meta.properties || {}
    meta.properties.syndication = syndications
    await hugo.saveEntry(postUri, { meta, content })
    await git.commit(`syndication on ${postUri}`)
  } catch (err) {
    debug('could not save syndication %s', err.stack)
    notify.sendError(err)
  }
}
