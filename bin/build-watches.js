#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const buildWatches = require('../src/build-watches')
const hugo = require('../src/services/hugo')(config.hugo)
const { join } = require('path')

;(async () => {
  console.log('Building watches...')

  const src = config.traktData
  const dst = join(config.hugo.dir, 'data/watches.json')

  console.log('  - Source:', src)
  console.log('  - Output:', dst)

  await buildWatches({ src, dst })

  const { meta, content } = await hugo.getEntry('/watches')
  meta.date = new Date()
  await hugo.saveEntry('/watches', { meta, content })

  console.log('built!')
})()
