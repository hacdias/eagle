#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const { hugo, webmentions } = require('../src/services')(config)
const { join } = require('path')
const fs = require('fs-extra')
const sha256 = require('../src/utils/sha256')

;(async () => {
  const redirs = webmentions._loadRedirects(join(hugo.publicDir, 'redirects.txt'))

  for (const [from, to] of Object.entries(redirs)) {
    const path = join(hugo.dataDir, 'mentions', sha256(from) + '.json')
    if (fs.existsSync(path)) {
      await fs.move(path, join(hugo.dataDir, 'mentions', sha256(to) + '.json'))
    }
  }
})()
