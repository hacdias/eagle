const debug = require('debug')('eagle:server:micropub')
const got = require('got')
const { extname } = require('path')
const { sha256 } = require('../../services/utils')
const { parse } = require('node-html-parser')

const getPhotos = async (meta, cdn) => {
  const photos = meta.properties.photo
  if (!photos || !Array.isArray(photos)) {
    return
  }

  const newPhotos = []
  let updated = false

  for (const photo of photos) {
    if (photo.startsWith('https://cdn.hacdias.com')) {
      newPhotos.push(photo)
      continue
    }

    debug('downloading %s', photo)

    try {
      const res = await got(photo, { responseType: 'buffer' })
      const hash = sha256(res.body)
      const ext = extname(photo)
      const filename = `images/${hash}${ext}`
      const url = await cdn.upload(res.body, filename)
      newPhotos.push(url)
      updated = true
    } catch (e) {
      newPhotos.push(photo)
      debug('could not download photo %s: %s', photo, e.stack)
    }
  }

  if (!updated) {
    return
  }

  return newPhotos
}

const getMentions = async (url, body) => {
  debug('will scrap %s for webmentions', url)
  const parsed = parse(body)

  const targets = parsed.querySelectorAll('.h-entry .e-content a')
    .map(p => p.attributes.href)
    .map(href => {
      try {
        const u = new URL(href, url)
        return u.href
      } catch (_) {
        return href
      }
    })

  debug('found webmentions: %o', targets)
  return targets
}

module.exports = {
  getPhotos,
  getMentions
}
