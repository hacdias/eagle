#!/usr/bin/env node
'use strict'

require('dotenv').config()

const csv = require('@fast-csv/format')
const fs = require('fs-extra')
const config = require('../src/config')()
const hugo = require('../src/services/hugo')(config.hugo)

async function outputBookmarks () {
  const stream = csv.format({
    headers: [
      'Title',
      'Date Added',
      'URL',
      'Tags'
    ]
  })
  stream.pipe(fs.createWriteStream('bookmarks.csv'))

  for await (const { meta } of hugo.getAll()) {
    if (!meta.properties) {
      continue
    }

    if (!meta.categories || !meta.categories.includes('bookmarks')) {
      continue
    }

    stream.write([meta.title, meta.date.getTime(), meta.properties['bookmark-of'], meta.tags])
  }

  stream.end()
}

async function outputReads () {
  const stream = csv.format({
    headers: [
      'Status',
      'Date',
      'Author',
      'Name',
      'Rating',
      'UID',
      'Tags'
    ]
  })
  stream.pipe(fs.createWriteStream('books.csv'))

  for await (const { meta } of hugo.getAll()) {
    if (!meta.properties) {
      continue
    }

    if (!meta.categories || !meta.categories.includes('reads')) {
      continue
    }

    stream.write([
      meta.properties['read-status'],
      meta.date.getTime(),
      meta.properties['read-of'][0].properties.author,
      meta.properties['read-of'][0].properties.name,
      meta.properties['read-of'][0].properties.rating || 0,
      meta.properties['read-of'][0].properties.uid,
      meta.properties['bookmark-of'],
      meta.tags
    ])
  }

  stream.end()
}

async function outputCheckins () {
  const stream = csv.format({
    headers: [
      'Date',
      'Name',
      'Country',
      'Region',
      'Locality',
      'Address',
      'Latitude',
      'Longitude',
      'Tags'
    ]
  })
  stream.pipe(fs.createWriteStream('checkins.csv'))

  for await (const { meta } of hugo.getAll({ keepOriginal: true })) {
    if (!meta.properties) {
      continue
    }

    if (!meta.categories || !meta.categories.includes('checkins')) {
      continue
    }

    stream.write([
      meta.date.getTime(),
      meta.properties.checkin.properties.name,
      meta.properties.checkin.properties['country-name'],
      meta.properties.checkin.properties.region,
      meta.properties.checkin.properties.locality,
      meta.properties.checkin.properties['street-address'],
      meta.properties.checkin.properties.latitude,
      meta.properties.checkin.properties.longitude,
      meta.tags
    ])
  }

  stream.end()
}

;(async () => {
  await outputBookmarks()
  await outputReads()
  await outputCheckins()
})()
