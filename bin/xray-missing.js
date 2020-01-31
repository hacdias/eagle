#!/usr/bin/env node
'use strict'

require('dotenv').config()

const { join } = require('path')

const config = require('../src/config')()
const hugo = require('../src/services/hugo')(config.hugo)

const xray = require('../src/services/xray')({
  domain: config.domain,
  twitter: config.twitter,
  entrypoint: config.xrayEntrypoint,
  dir: join(hugo.dataDir, 'xray')
})

;(async () => {
  for await (const { meta } of hugo.getAll()) {
    if (!meta.properties) {
      continue
    }

    for (const type of ['like-of', 'repost-of', 'in-reply-to']) {
      if (!meta.properties[type]) {
        continue
      }

      await xray.requestAndSave(meta.properties[type][0])
    }
  }
})()
