#!/usr/bin/env node
'use strict'

require('dotenv').config()

/*

FIXME ?

const config = require('../src/config')()
const hugo = require('../src/services/hugo')(config.hugo)
const cdn = require('../src/services/bunnycdn')(config.bunny)
const { getPhotos } = require('../src/server/micropub/helpers')

;(async () => {
  for await (const { post, meta, content } of hugo.getAll()) {
    if (!meta.properties) {
      continue
    }

    try {
      const newPhotos = await getPhotos(meta, cdn)

      if (newPhotos) {
        meta.properties.photo = newPhotos
        await hugo.saveEntry(post, { meta, content })
      }
    } catch (e) {
      console.error('could not update post %s: %s', post, e.stack)
    }
  }
})() */
