#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const { hugo, xray } = require('../src/services')(config)

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
