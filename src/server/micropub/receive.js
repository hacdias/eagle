const debug = require('debug')('eagle:server:micropub')
const execa = require('execa')
const { parse } = require('node-html-parser')
const transformer = require('./transformer')
const syndicate = require('./syndicate')

async function reloadCaddy () {
  try {
    await execa('pkill', ['-USR1', 'caddy'])
    debug('caddy config reloaded')
  } catch (e) {
    debug('could not reload caddy config: %s', e.stack)
  }
}

const getMentions = async (url, body) => {
  debug('will scrap %s for webmentions', url)
  const parsed = parse(body)

  const targets = parsed.querySelectorAll('.h-entry .e-content a')
    .map(p => p.attributes.href)
    .map(href => {
      try {
        const u = new URL(href, url)
        return u.href
      } catch (_) {
        return href
      }
    })

  debug('found webmentions: %o', targets)
  return targets
}

const sendWebmentions = async (post, url, related, services) => {
  const { hugo, notify, webmentions } = services
  const targets = [...related]

  try {
    const html = await hugo.getEntryHTML(post)
    const mentions = await getMentions(url, html)
    targets.push(...mentions)
  } catch (err) {
    notify.sendError(err)
  }

  try {
    await webmentions.send({ source: url, targets })
  } catch (err) {
    notify.sendError(err)
  }
}

module.exports = ({ services, domain }) => {
  const { xray, notify, hugo, git } = services

  return async (req, res, data) => {
    const postData = transformer.createPost(data)

    // Fetch all related URLs XRay. Fail silently.
    await Promise.all(postData.syndication.related
      .map(url => xray.requestAndSave(url).catch(notify.sendError))
    )

    const { post } = await hugo.newEntry(postData)
    const url = `${domain}${post}`

    res.redirect(202, url)

    await git.commit(`add ${post}`)
    await hugo.build()

    // Reload caddy config asynchronously if there are any aliases
    // so it can load the redirects.
    if (postData.meta.aliases) {
      reloadCaddy()
    }

    // Async operations
    syndicate(services, post, url, postData)
    sendWebmentions(post, url, postData.syndication.related, services)
  }
}
