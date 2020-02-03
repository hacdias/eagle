const debug = require('debug')('eagle:server:micropub')
const got = require('got')
const { extname } = require('path')
const { sha256 } = require('../../services/utils')

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

module.exports = {
  getPhotos
}
