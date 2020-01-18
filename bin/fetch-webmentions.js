#!/usr/bin/env node
'use strict'

require('dotenv').config()

const got = require('got')
const eagle = require('../src/eagle').fromEnvironment()

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

;(async () => {
  let mentions

  for (let i = 0; (mentions = await getWebmentions(i)).length > 0; i++) {
    for (const mention of mentions) {
      eagle.receiveWebMention({ post: mention, target: mention['wm-target'] }, { skipGit: true, skipBuild: true })
    }
  }
})()
