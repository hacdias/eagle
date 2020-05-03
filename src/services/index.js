const path = require('path')

const { default: PQueue } = require('p-queue')

const createTwitter = require('./twitter')
const createXRay = require('./xray')
const createCdn = require('./cdn')
const createGit = require('./git')
const createHugo = require('./hugo')
const createWebmentions = require('./webmentions')
const createNotify = require('./notify')
const createActivityPub = require('./activitypub')

module.exports = function getServices (config) {
  const queue = new PQueue({
    concurrency: 1,
    autoStart: true
  })

  const twitter = createTwitter({
    apiKey: config.twitter.apiKey,
    apiSecret: config.twitter.apiSecret,
    accessToken: config.twitter.accessToken,
    accessTokenSecret: config.twitter.accessTokenSecret
  })

  const hugo = createHugo({
    dir: config.hugo.dir,
    publicDir: config.hugo.publicDir
  })

  const xray = createXRay({
    apiEndpoint: config.xrayEndpoint,
    storeDir: path.join(hugo.dataDir, 'xray'),
    twitterConf: config.twitter,
    defaultDomain: config.domain
  })

  const cdn = createCdn({
    zone: config.bunny.zone,
    key: config.bunny.key,
    base: config.bunny.base
  })

  const git = createGit({
    cwd: hugo.dir
  })

  const webmentions = createWebmentions({
    redirectsFile: path.join(hugo.publicDir, 'redirects.txt'),
    storeDir: path.join(hugo.dataDir, 'mentions'),
    telegraphToken: config.telegraphToken,
    domain: config.domain,
    git,
    cdn
  })

  const notify = createNotify({
    telegramChatId: config.telegram.chatId,
    telegramToken: config.telegram.token
  })

  const activitypub = createActivityPub({
    hugo,
    webmentions,
    queue,
    domain: config.domain,
    store: config.activityPub.store
  })

  return Object.freeze({
    twitter,
    xray,
    cdn,
    git,
    hugo,
    webmentions,
    notify,
    queue,
    activitypub
  })
}
