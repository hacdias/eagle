const ar = require('../utils/ar')
const express = require('express')

module.exports = ({ services: { activitypub } }) => {
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
    res.sendStatus(await activitypub.inboxHandler(req.body))
  }))

  router.get('/outbox', ar(async (_, res) => {
    const outbox = await activitypub.outboxHandler()
    res.json(outbox)
  }))

  router.post('/outbox', (_, res) => {
    res.sendStatus(501)
  })

  return router
}
