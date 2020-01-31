#!/usr/bin/env node
'use strict'

require('dotenv').config()
const got = require('got')
const { join } = require('path')
const config = require('../src/config')()

const getWebmentions = async (page) => {
  const { body } = await got('https://webmention.io/api/mentions.jf2', {
    searchParams: {
      token: process.env.WEBMENTION_IO_TOKEN,
      page
    },
    responseType: 'json'
  })

  return body.children
}

const hugo = require('../src/services/hugo')(config.hugo)

const git = require('../src/services/git')({
  cwd: hugo.dir
})

const webmentions = require('../src/services/webmentions')({
  token: config.telegraphToken,
  domain: config.domain,
  dir: join(hugo.dataDir, 'mentions'),
  git
})

;(async () => {
  let mentions

  for (let i = 0; (mentions = await getWebmentions(i)).length > 0; i++) {
    for (const mention of mentions) {
      await webmentions.receive({
        post: mention,
        target: mention['wm-target']
      })
    }
  }
})()
