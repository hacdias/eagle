require('dotenv').config()

const { join } = require('path')
const express = require('express')
const { default: PQueue } = require('p-queue')

const config = require('./config')()

/* CONFIGURE SERVICES */

const hugo = require('./services/hugo')({
  ...config.hugo,
  domain: config.domain
})

const xray = require('./services/xray')({
  domain: config.domain,
  twitter: config.twitter,
  entrypoint: config.xrayEntrypoint,
  dir: join(hugo.dataDir, 'xray')
})

const git = require('./services/git')({
  cwd: hugo.dir
})

const webmentions = require('./services/webmentions')(config.telegraphToken)

const telegram = require('./services/telegram')({
  ...config.telegram,
  git,
  hugo
})

const posse = require('./services/posse')({
  twitter: require('./services/twitter')(config.twitter)
})

const queue = new PQueue({
  concurrency: 1,
  autoStart: true
})

/* SETUP EXPRESS APP */

const app = express()

app.use(express.json())

app.use('/micropub', require('./routes/micropub')({
  domain: config.domain,
  xray,
  webmentions,
  posse,
  hugo,
  git,
  telegram,
  queue,
  tokenReference: config.tokenReference
}))

app.use('/webmention', require('./routes/webmention')({
  secret: process.env.WEBMENTION_IO_WEBHOOK_SECRET,
  dir: join(hugo.dataDir, 'mentions'),
  domain: config.domain,
  git,
  hugo,
  telegram,
  queue
}))

app.get('/now', require('./routes/now')())

app.get('/robots.txt', (_, res) => {
  res.header('Content-Type', 'text/plain')
  res.send('UserAgent: *\nDisallow: /')
})

app.use((_, res) => {
  res.header('Content-Type', 'text/plain')
  res.status(404).send("Darlings, there's nothing to see here! Muah 💋")
})

app.use((err, req, res, next) => {
  require('debug')('eagle')(err.stack)
  telegram.sendError(err)
  res.status(500).send('Something broke!')
})

app.listen(config.port, () => console.log(`Listening on port ${config.port}!`))
