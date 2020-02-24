#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const buildWatches = require('../src/build-watches')
const { join } = require('path')

;(async () => {
  console.log('Building watches...')

  const source = config.traktData
  const output = join(config.hugo.dir, 'data/watches.json')

  console.log('  - Source:', source)
  console.log('  - Output:', output)

  await buildWatches({ source, output })

  console.log('built!')
})()
