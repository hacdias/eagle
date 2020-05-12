#!/usr/bin/env node
'use strict'

require('dotenv').config()

const got = require('got')
const config = require('../src/config')()
const { webmentions } = require('../src/services')(config)

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
      const url = mention.url || mention['wm-source']

      if (url.startsWith('https://ownyourswarm.p3k.io/')) {
        continue
      }

      await webmentions.receive({
        post: mention,
        target: mention['wm-target']
      }, true)
    }
  }
})()
