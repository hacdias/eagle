#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const kb = require('../src/services/kb')

;(async () => {
  const getKB = kb({
    hugoDir: config.hugo.dir,
    notesRepoDir: config.notesRepo
  })

  await getKB()
})()
