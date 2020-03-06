#!/usr/bin/env node
'use strict'

require('dotenv').config()

const { extname } = require('path')
const crypto = require('crypto')
const config = require('../src/config')()
const cdn = require('../src/services/bunnycdn')(config.bunny)
const sharp = require('sharp')
const fs = require('fs-extra')

function sha256 (data) {
  return crypto.createHash('sha256').update(data).digest('hex')
}

const matrix = {
  jpeg: [
    [1000],
    [2000],
    [600, 400]
  ],
  webp: [
    [1000],
    [2000],
    [600, 400]
  ]
}

;(async () => {
  const photos = process.argv.slice(2, process.argv.length)
  if (photos.length === 0) {
    console.log('No file')
    process.exit(1)
  }

  for (const file of photos) {
    console.log(file)
    if (!['.jpg', '.jpeg'].includes(extname(file))) {
      console.log('Unsupported file type')
      process.exit(1)
    }

    const buff = await fs.readFile(file)
    const hash = sha256(file)

    console.log('\t', await cdn.upload(buff, `photos/${hash}.jpeg`))

    for (const type in matrix) {
      for (const sizes of matrix[type]) {
        const trans = sharp(buff)[type]().resize(...sizes)
        const filename = `${hash}_${sizes.join('x')}${sizes.length === 1 ? 'x' : ''}.${type}`
        await fs.outputFile(filename, await trans.toBuffer())
        console.log('\t', await cdn.upload(trans, `photos/${filename}`))
      }
    }
  }
})()
