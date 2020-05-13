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

  if (type === 'notes' || type === 'article') {
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

module.exports = async function syndicate (services, postUri, postUrl, postData) {
  const { twitter, hugo, git, notify } = services
  const { type } = postData
  const { targets, related } = postData.syndication

  const syndications = (await Promise.all([
    ...targets.map(async service => {
      try {
        if (service === 'twitter') {
          return sendToTwitter({ type, postData, postUrl, twitter })
        }
      } catch (err) {
        debug('syndication failed to service %s: %s', service, err.stack)
        notify.sendError(err)
      }

      debug('syndication to %s does not exist', service)
    }),
    ...related.map(async url => {
      try {
        if (isTwitterURL(url)) {
          return sendToTwitter({ type, postData, postUrl, twitter, url })
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
