const { join } = require('path')
const fs = require('fs-extra')
const actors = require('./actor')
const requests = require('./req')

const getCategory = async (publicDir, category) => {
  const { items } = await fs.readJSON(join(publicDir, category, 'feed.json'))

  return Promise.all(items.map(({ url }) => {
    const id = new URL(url).pathname
    return fs.readJSON(join(publicDir, id, 'index.as2'))
  }))
}

module.exports = function createActivityPub ({ domain, hugo, queue, webmentions, store }) {
  fs.ensureDirSync(store)

  const self = new URL(domain).origin + '/'
  const privateKey = fs.readFileSync(join(store, 'private.key'), 'ascii')

  const backupFile = join(store, 'backup.json')
  const followersFile = join(store, 'followers.json')

  const followers = fs.existsSync(followersFile)
    ? fs.readJSON(followersFile)
    : {}

  const create = async (data) => {
    if (!data.object) {
      return 400
    }

    const replyTo = data.object.inReplyTo
    const id = data.object.id

    if (typeof replyTo !== 'string' || typeof id !== 'string') {
      return 400
    }

    await webmentions.send({
      source: id,
      targets: [replyTo]
    })

    return 201
  }

  const follow = async (data) => {
    const { url, inbox } = await actors.get(data.actor)

    if (!followers[url]) {
      followers[url] = inbox
      await fs.writeJSON(followersFile, followers)
    }

    delete data['@context']

    const accept = {
      '@context': 'https://www.w3.org/ns/activitystreams',
      to: data.actor,
      id: self + require('uuid').v1(),
      actor: self,
      object: data,
      type: 'Accept'
    }

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

  return Object.freeze({
    outboxHandler,
    inboxHandler
  })
}
