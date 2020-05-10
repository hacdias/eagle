#!/usr/bin/env node
'use strict'

require('dotenv').config()

const { join } = require('path')
const config = require('../src/config')()
const buildKB = require('../src/builders/kb')

;(async () => {
  const src = join(config.notes.repositoryDir)
  const dst = join('', 'content')

  console.log('Building knowledge base...')
  console.log('  - Source:', src)
  console.log('  - Output:', dst)

  await buildKB({ src, dst })
  console.log('built!')
})()
