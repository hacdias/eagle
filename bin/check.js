#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const { hugo, xray } = require('../src/services')(config)

;(async () => {
  const list = []

  for await (const { meta } of hugo.getAll()) {
    list.push(...Object.keys(meta))
  }

  console.log(new Set(list))
})()
