#!/usr/bin/env node
'use strict'

require('dotenv').config()

const meow = require('meow')
const fs = require('fs-extra')
const config = require('../src/config')()
const hugo = require('../src/services/hugo')(config.hugo)

const createBookmarks = require('../src/csv/bookmarks')
const createReads = require('../src/csv/reads')

const cli = meow(`
  Usage
    $ csv <type>

  Options
    --output, -o  output file
`, {
  flags: {
    output: {
      type: 'string',
      alias: 'o',
      default: 'out.csv'
    }
  }
})

const fns = {
  bookmarks: createBookmarks,
  reads: createReads
}

;(async () => {
  const type = cli.input ? cli.input.join(' ') : ''
  const output = cli.flags.output

  if (!type || !output || Array.isArray(output)) {
    return cli.showHelp()
  }

  const out = output === 'stdout'
    ? process.stdout
    : fs.createWriteStream(output)

  const fn = fns[type]
  if (fn) {
    await fn(out, hugo)
  } else {
    throw new Error('invalid type')
  }
})()
