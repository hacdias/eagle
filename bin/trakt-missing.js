#!/usr/bin/env node
'use strict'

require('dotenv').config()
const got = require('got')

const config = require('../src/config')()
const hugo = require('../src/services/hugo')(config.hugo)

;(async () => {
  const ids = []

  for await (const { meta } of hugo.getAll({ keepOriginal: true })) {
    if (!meta.properties || !meta.categories || !meta.categories.includes('watches')) {
      continue
    }

    ids.push(meta.properties['watch-of'].properties['trakt-id'])
  }

  const { body } = await got('https://api.trakt.tv/sync/history?limit=2000', {
    headers: {
      Authorization: `Bearer ${config.trakt.token}`,
      'Content-Type': 'application/json',
      Accept: 'application/json',
      'trakt-api-key': config.trakt.clientID,
      'trakt-api-version': '2'
    },
    responseType: 'json'
  })

  for (const item of body) {
    if (ids.includes(item.id)) {
      continue
    }

    const watch = {
      'trakt-id': item.id
    }

    if (item.type === 'episode') {
      watch.title = item.episode.title
      watch.season = item.episode.season
      watch.episode = item.episode.number
      watch.ids = item.episode.ids
      watch.url = `https://trakt.tv/shows/${item.show.ids.slug}/seasons/${item.episode.season}/episodes/${item.episode.season}`

      watch.show = {
        type: 'h-card',
        properties: {
          title: item.show.title,
          year: item.show.year,
          url: `https://trakt.tv/shows/${item.show.ids.slug}`,
          ids: item.show.ids
        }
      }
    } else {
      watch.title = item.movie.title
      watch.year = item.movie.year
      watch.url = `https://trakt.tv/movies/${item.movie.ids.slug}`
      watch.ids = item.movie.ids
    }

    const meta = {
      categories: ['watches'],
      date: new Date(item.watched_at),
      properties: {
        'watch-of': {
          type: 'h-card',
          properties: watch
        }
      }
    }

    await hugo.newEntry({ meta, content: '', slug: '' }, { keepOriginal: true })
  }
})()
