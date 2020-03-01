const { join } = require('path')
const express = require('express')
const { default: PQueue } = require('p-queue')
const debug = require('debug')('eagle:server')

const config = require('../config')()

module.exports = function () {
  // Configure services

  const cdn = require('../services/bunnycdn')(config.bunny)

  const hugo = require('../services/hugo')({
    ...config.hugo,
    domain: config.domain
  })

  const xray = require('../services/xray')({
    domain: config.domain,
    twitter: config.twitter,
    entrypoint: config.xrayEntrypoint,
    dir: join(hugo.dataDir, 'xray')
  })

  const git = require('../services/git')({
    cwd: hugo.dir
  })

  const webmentions = require('../services/webmentions')({
    token: config.telegraphToken,
    domain: config.domain,
    dir: join(hugo.dataDir, 'mentions'),
    git,
    hugo,
    cdn
  })

  const notify = require('../services/notify')(config.telegram)

  const posse = require('../services/posse')({
    twitter: require('../services/twitter')(config.twitter)
  })

  const queue = new PQueue({
    concurrency: 1,
    autoStart: true
  })

  const activitypub = require('../services/activitypub')({
    queue,
    hugo,
    webmentions,
    domain: config.domain,
    store: config.activityPub.store
  })

  // Start bot only on production...
  if (process.env.NODE_ENV === 'production') {
    require('../services/bot')({
      ...config.telegram,
      git,
      hugo
    })
  }

  // Setup express app
  const app = express()

  app.use(express.json())

  app.use('/micropub', require('./micropub')({
    domain: config.domain,
    tokenReference: config.tokenReference,
    xray,
    webmentions,
    posse,
    hugo,
    git,
    notify,
    queue,
    cdn
  }))

  app.use('/webmention', require('./webmention')({
    secret: process.env.WEBMENTION_IO_WEBHOOK_SECRET,
    webmentions,
    hugo,
    notify,
    queue
  }))

  app.get('/now', require('./now')())

  app.get('/webfinger', require('./webfinger')({
    domain: config.domain,
    user: config.activityPub.user
  }))

  app.use('/activitypub', require('./activitypub')({
    activitypub
  }))

  app.post('/notes', require('./hook-notes')({
    git,
    hugo,
    queue,
    notesRepo: config.notesRepo,
    secret: config.notesSecret
  }))

  app.post('/repo', require('./hook-repo')({
    git,
    hugo,
    queue,
    secret: config.hookSecret
  }))

  app.get('/build/watches', require('./build-watches')({
    git,
    hugo,
    secret: config.traktSecret,
    queue,
    source: config.traktData
  }))

  app.get('/robots.txt', (_, res) => {
    res.header('Content-Type', 'text/plain')
    res.send('UserAgent: *\nDisallow: /')
  })

  app.use((_, res) => {
    res.header('Content-Type', 'text/plain')
    res.status(404).send("Darlings, there's nothing to see here! Muah 💋")
  })

  app.use((err, req, res, next) => {
    debug(err.stack)
    notify.sendError(err)

    if (!res.headersSent) {
      res.sendStatus(500)
    }
  })

  app.listen(config.port, () => console.log(`Listening on http://127.0.0.1:${config.port}`))
}
