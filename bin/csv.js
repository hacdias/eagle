#!/usr/bin/env node
'use strict'

require('dotenv').config()

const meow = require('meow')
const fs = require('fs-extra')
const config = require('../src/config')()
const hugo = require('../src/services/hugo')(config.hugo)

const createBookmarks = require('../src/csv/bookmarks')
const createCheckins = require('../src/csv/checkins')
const createReads = require('../src/csv/reads')
const createWatches = require('../src/csv/watches')

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

;(async () => {
  const type = cli.input ? cli.input.join(' ') : ''
  const output = cli.flags.output

  if (!type || !output || Array.isArray(output)) {
    return cli.showHelp()
  }

  const out = output === 'stdout'
    ? process.stdout
    : fs.createWriteStream(output)

  switch (type) {
    case 'bookmarks':
      await createBookmarks(out, hugo)
      break
    case 'checkins':
      await createCheckins(out, hugo)
      break
    case 'reads':
      await createReads(out, hugo)
      break
    case 'watches':
      await createWatches(out, hugo)
      break
    default:
      throw new Error('invalid type')
  }

  console.log(type, output)
})()
