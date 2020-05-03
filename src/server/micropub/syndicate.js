const debug = require('debug')('eagle:syndicate')

async function sendToTwitter ({ url, type, status, twitter }) {
  const id = url ? new URL(url).pathname.split('/').pop() : null
  let res

  switch (type) {
    case 'notes':
      res = await twitter.tweet({ status })
      break
    case 'likes':
      await twitter.like(id)
      break
    case 'reposts':
      res = await twitter.retweet(id)
      break
    case 'replies':
      res = await twitter.tweet({ status, inReplyTo: id })
      break
    default:
      break
  }

  if (res) {
    return `https://twitter.com/hacdias/status/${res.id_str}`
  }
}

const isTwitterURL = url => url.startsWith('https://twitter.com')

module.exports = async function syndicate (services, postUri, postUrl, postData, commands) {
  commands = commands || {}
  commands['mp-syndicate-to'] = commands['mp-syndicate-to'] || []

  const { twitter, hugo, git, notify } = services
  const { type } = postData

  const status = postData.content.length <= 280
    ? postData.content
    : `${postData.content.substr(0, 230).trim()}... ${postUrl}`

  const syndications = await Promise.all([
    ...commands['mp-syndicate-to'].map(async service => {
      try {
        if (service === 'twitter') {
          return sendToTwitter({ type: 'notes', status, twitter })
        }
      } catch (err) {
        debug('syndication failed to service %s: %s', service, err.stack)
        notify.sendError(err)
      }

      debug('syndication to %s does not exist', service)
    }),
    ...postData.relates.map(async url => {
      try {
        if (isTwitterURL(url)) {
          return sendToTwitter({ url, type, status, twitter })
        }
      } catch (err) {
        debug('syndication failed to service %s: %s', url, err.stack)
        notify.sendError(err)
      }

      debug('syndication to %s unknown', url)
    })
  ]).filter(url => !!url)

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
