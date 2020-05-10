const debug = require('debug')('eagle:server')
const express = require('express')

const getServices = require('../services')
const getConfig = require('../config')

const createMicropub = require('./micropub')
// const createWebHookNotes = require('./webhook-notes')
const createWebHookWebsite = require('./webhook-website')
const createBuildWatches = require('./build-watches')
const createWebmention = require('./webmention')
const createWebfinger = require('./webfinger')
const createActivityPub = require('./activitypub')
const createBot = require('./bot')

module.exports = function () {
  const config = getConfig()
  const services = getServices(config)
  const app = express()

  app.use(express.json())

  app.use('/micropub', createMicropub({
    services,
    domain: config.domain,
    tokenReference: config.tokenReference
  }))

  // app.post('/webhooks/notes', createWebHookNotes({
  //   services,
  //   repositoryDir: config.notes.repositoryDir,
  //   secret: config.notes.hookSecret
  // }))

  app.post('/webhooks/website', createWebHookWebsite({
    services,
    secret: config.website.hookSecret
  }))

  app.get('/build/watches', createBuildWatches({
    services,
    repositoryDir: config.trakt.repositoryDir,
    secret: config.trakt.secret
  }))

  app.use('/webmention', createWebmention({
    secret: config.webmentionIoSecret,
    services
  }))

  app.get('/webfinger', createWebfinger({
    domain: config.domain,
    user: config.activityPub.user
  }))

  app.use('/activitypub', createActivityPub({
    services
  }))

  app.use((err, req, res, next) => {
    debug(err.stack)
    services.notify.sendError(err)

    if (!res.headersSent) {
      res.sendStatus(500)
    }
  })

  // Start bot only on production...
  if (process.env.NODE_ENV === 'production') {
    createBot({
      telegramChatId: config.telegram.chatId,
      telegramToken: config.telegram.token,
      services
    })

    console.log('Telegram bot started!')
  }

  app.listen(config.port, () => console.log(`Listening on http://127.0.0.1:${config.port}`))
}
