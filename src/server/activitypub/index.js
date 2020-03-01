const { ar } = require('../utils')
const { join } = require('path')
const fs = require('fs-extra')
const express = require('express')
const actors = require('./actor')
const crypto = require('crypto')
const got = require('got')

const getCategory = async (publicDir, category) => {
  const { items } = await fs.readJSON(join(publicDir, category, 'feed.json'))

  return Promise.all(items.map(({ url }) => {
    const id = new URL(url).pathname
    return fs.readJSON(join(publicDir, id, 'index.as2'))
  }))
}

module.exports = ({ hugo, queue, webmentions, store }) => {
  fs.ensureDirSync(store)

  const backup = join(store, 'backup.json')
  const followers = join(store, 'followers.json')

  const privateKey = fs.readFileSync(join(store, 'private.key'), 'ascii')

  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json({
    type: [
      'application/ld+json',
      'application/activity+json',
      'application/json'
    ]
  }))

  router.use(express.urlencoded({ extended: true }))

  router.get('/inbox', ar(async (req, res) => {
    res.sendStatus(501)
  }))

  const create = async (req, res) => {
    if (!req.body.object) {
      return res.sendStatus(400)
    }

    const replyTo = req.body.object.inReplyTo
    const id = req.boy.object.id

    if (typeof replyTo !== 'string' || typeof id !== 'string') {
      return res.sendStatus(400)
    }

    await webmentions.send({
      source: id,
      targets: [replyTo]
    })

    return res.sendStatus(201)
  }

  const follow = async (req, res) => {
    const follower = await actors.get(req.body.actor)

    await fs.appendFile(followers, JSON.stringify(follower) + '\n')

    delete req.body['@context']

    const accept = {
      '@context': 'https://www.w3.org/ns/activitystreams',
      to: req.body.actor,
      id: require('uuid').v1(),
      actor: follower.url,
      object: req.body,
      type: 'Accept'
    }

    const inbox = new URL(follower.inbox)
    const signer = crypto.createSign('sha256')
    const date = new Date()

    const body = JSON.stringify(accept)
    const digest = 'SHA-256=' + crypto.createHash('sha256').update(body).digest('base64')

    const stringToSign = `(request-target): post ${inbox.pathname}\nhost: ${inbox.host}\ndate: ${date.toUTCString()}\ndigest: ${digest}`
    signer.update(stringToSign)
    signer.end()
    const signature = signer.sign(privateKey).toString('base64')

    const header = `keyId="https://hacdias.com/#key",algorithm="rsa-sha256",headers="(request-target) host date digest",signature="${signature}"`

    await got.post(inbox.href, {
      body,
      headers: {
        'Content-Type': 'application/activity+json',
        Host: inbox.host,
        Date: date.toUTCString(),
        Digest: digest,
        Signature: header
      }
    })

    console.log('SENT')

    return res.sendStatus(200)
  }

  router.post('/inbox', ar(async (req, res) => {
    await fs.appendFile(backup, JSON.stringify(req.body) + '\n')

    switch (req.body.type) {
      case 'Create':
        return create(req, res)
      case 'Follow':
        return follow(req, res)
      default:
        return res.sendStatus(501)
    }
  }))

  router.get('/outbox', ar(async (req, res) => {
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

    res.json({
      '@context': 'https://www.w3.org/ns/activitystreams',
      id: 'https://www.w3.org/ns/activitystreams#Public',
      summary: "Henrique's Posts",
      type: 'OrderedCollection',
      totalItems: posts.length,
      orderedItems: posts
    })
  }))

  router.post('/outbox', (req, res) => {
    res.sendStatus(501)
  })

  return router
}
