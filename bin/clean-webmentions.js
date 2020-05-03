#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const { hugo } = require('../src/services')(config)
const fs = require('fs-extra')
const path = require('path')

;(async () => {
  const dir = path.join(hugo.dataDir, 'mentions')
  const files = fs.readdirSync(dir)

  for (const file of files) {
    const filePath = path.join(dir, file)
    const mentions = fs.readJSONSync(filePath)

    if (mentions.length === 0) {
      fs.removeSync(filePath)
      continue
    }

    const swarm = mentions.filter(m => m.author.name === 'Swarm')
    if (swarm.length) {
      fs.removeSync(filePath)
    }
  }
})()
