const debug = require('debug')('eagle:syndicate')
const createTweets = require('./create-tweets')

function smallStatus (content, url) {
  return content.length <= 280
    ? content
    : `${content.substr(0, 230).trim()}... ${url}`
}

const twitterAllowedTypes = Object.freeze([
  'replies', 'articles', 'notes'
])

async function sendToTwitter ({ url, type, postData, postUrl, twitter }) {
  if (!twitterAllowedTypes.includes(type)) {
    throw new Error('invalid type for twitter syndication' + type)
  }

  if (type === 'articles') {
    // An article is the simplest case because I don't want to dump
    debug('post is article, posting short status')
    // the entire text on Twitter. So, in this case, we just publish
    // a short bit and return that link. However, I rarely publish articles
    // using Micropub anyways.
    return [await twitter.tweet({
      status: smallStatus(postData.content, postUrl)
    })]
  }

  debug('post is reply or note, posting entire content')

  const tweets = await createTweets(postData.content, postUrl)
  const links = []

  let prev = null
  for (let i = 0; i < tweets.length; i++) {
    const opts = {
      status: tweets[i]
    }

    if (i === 0) {
      if (type === 'replies') {
        if (postData.modifiers.includes('+RT')) {
          opts.attachment = url
        } else {
          opts.inReplyTo = new URL(url).pathname.split('/').pop()
        }
      }
    } else {
      opts.inReplyTo = prev
    }

    const res = await twitter.tweet(opts)
    prev = res.id_str
    links.push(`https://twitter.com/hacdias/status/${prev}`)
  }

  return links
}

const isTwitterURL = url => url.startsWith('https://twitter.com')

module.exports = async function syndicate (services, postUri, postUrl, postData) {
  const { twitter, hugo, git, queue, notify } = services
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
      return []
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

      return []
    })
  ])).reduce((prev, curr) => prev.concat(curr), [])

  if (syndications.length === 0) {
    return
  }

  await queue.add(async () => {
    try {
      const { meta, content } = await hugo.getEntry(postUri)
      meta.properties = meta.properties || {}
      meta.properties.syndication = syndications
      await hugo.saveEntry(postUri, { meta, content })
      await git.commit(`syndication on ${postUri}`)
      await hugo.build()
    } catch (err) {
      debug('could not save syndication %s', err.stack)
      notify.sendError(err)
    }
  })
}

module.exports.sendToTwitter = sendToTwitter
