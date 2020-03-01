const { ar } = require('./utils')
const { join } = require('path')
const fs = require('fs-extra')
const express = require('express')

const getCategory = async (publicDir, category) => {
  const { items } = await fs.readJSON(join(publicDir, category, 'feed.json'))

  return Promise.all(items.map(({ url }) => {
    const id = new URL(url).pathname
    return fs.readJSON(join(publicDir, id, 'index.as2'))
  }))
}

module.exports = ({ hugo, queue, backupFile }) => {
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

  router.post('/inbox', ar(async (req, res) => {
    await fs.appendFile(backupFile, JSON.stringify(req.body) + '\n')

    switch (req.body.type) {
      case 'Follow':
        console.log('Follow request')
        break
      case 'Undo':
        console.log('Undo')
        break
      case 'Create':
        console.log('Create')
        break
      default:
        return res.sendStatus(404)
    }

    res.sendStatus(501)
  }))

  router.get('/outbox', ar(async (req, res) => {
    const posts = await queue.add(async () => (
      await Promise.all([
        await getCategory(hugo.publicDir, 'notes'),
        await getCategory(hugo.publicDir, 'articles'),
        await getCategory(hugo.publicDir, 'replies')
      ])
    ).reduce((acc, curr) => {
      acc.push(...curr)
      return acc
    }, [])
      .sort((a, b) => new Date(b.published) - new Date(a.published))
    )

    res.json({
      '@context': 'https://www.w3.org/ns/activitystreams',
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
