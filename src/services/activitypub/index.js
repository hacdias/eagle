const { join } = require('path')
const fs = require('fs-extra')
const { v4: uuidv4 } = require('uuid')
const actors = require('./actor')
const requests = require('./req')
const debug = require('debug')('eagle:activity')

const getCategory = async (publicDir, category) => {
  const { items } = await fs.readJSON(join(publicDir, category, 'feed.json'))

  return Promise.all(items.map(({ url }) => {
    const id = new URL(url).pathname
    return fs.readJSON(join(publicDir, id, 'index.as2'))
  }))
}

module.exports = function createActivityPub ({ hugo, webmentions, queue, domain, store }) {
  fs.ensureDirSync(store)

  const self = new URL(domain).origin + '/'
  const privateKey = fs.readFileSync(join(store, 'private.key'), 'ascii')

  const backupFile = join(store, 'backup.json')
  const followersFile = join(store, 'followers.json')

  const followers = fs.existsSync(followersFile)
    ? fs.readJSON(followersFile)
    : {}

  const create = async (data) => {
    debug('got create request')
    if (!data.object) {
      debug('object not present')
      return 400
    }

    const replyTo = data.object.inReplyTo
    const id = data.object.id

    if (typeof replyTo !== 'string' || typeof id !== 'string') {
      debug('invalid reply %s or %id', replyTo, id)
      return 400
    }

    debug('creating webmention from %s to %s', id, replyTo)
    await webmentions.send({
      source: id,
      targets: [replyTo]
    })

    return 201
  }

  const follow = async (data) => {
    debug('follow request')
    const { url, inbox } = await actors.get(data.actor)

    if (!followers[url]) {
      debug('new follower: %s', url)
      followers[url] = inbox
      await fs.writeJSON(followersFile, followers)
    }

    delete data['@context']

    const accept = {
      '@context': 'https://www.w3.org/ns/activitystreams',
      to: data.actor,
      id: self + uuidv4(),
      actor: self,
      object: data,
      type: 'Accept'
    }

    debug('sending accept')
    await requests.sendSigned(privateKey, accept, inbox)
    return 200
  }

  const inboxHandler = async (data) => {
    data.handled = false
    let status = 502

    try {
      switch (data.type) {
        case 'Create':
          status = create(data)
          data.handled = true
          break
        case 'Follow':
          status = follow(data)
          data.handled = true
          break
      }
    } finally {
      await fs.appendFile(backupFile, JSON.stringify(data) + '\n')
    }

    return status
  }

  const outboxHandler = async () => {
    const posts = (await queue.add(async () => Promise.all([
      await getCategory(hugo.publicDir, 'notes'),
      await getCategory(hugo.publicDir, 'articles'),
      await getCategory(hugo.publicDir, 'replies')
    ])))
      .reduce((acc, curr) => {
        acc.push(...curr)
        return acc
      }, [])
      .sort((a, b) => new Date(b.published) - new Date(a.published))
      .map(item => {
        return {
          id: item.id,
          type: 'Create',
          actor: item.attributedTo,
          published: item.published,
          to: item.to,
          cc: item.cc,
          object: item
        }
      })

    return {
      '@context': 'https://www.w3.org/ns/activitystreams',
      id: `${self}activitypub/outbox`,
      summary: "Henrique's Posts",
      type: 'OrderedCollection',
      totalItems: posts.length,
      orderedItems: posts
    }
  }

  const postArticle = async (permalink) => {
    debug('posting %s', permalink)
    const item = await fs.readJSON(join(hugo.publicDir, permalink, 'index.as2'))

    const post = {
      '@context': ['https://www.w3.org/ns/activitystreams'],
      id: item.id,
      type: 'Create',
      actor: item.attributedTo,
      published: item.published,
      to: item.to,
      object: item
    }

    for (const inbox of Object.values(followers)) {
      try {
        await requests.sendSigned(privateKey, post, inbox)
      } catch (e) {
        debug('failed to send %s to %s', permalink, inbox)
      }
    }

    debug('posting %s done', permalink)
  }

  return Object.freeze({
    postArticle,
    outboxHandler,
    inboxHandler
  })
}
