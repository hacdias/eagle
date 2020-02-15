const express = require('express')
const debug = require('debug')('eagle:server:micropub')
const multer = require('multer')
const mime = require('mime-types')
const { ar } = require('../utils')
const helpers = require('./helpers')

const { extname } = require('path')
const { sha256 } = require('../../services/utils')
const { parseJson, parseFormEncoded } = require('@hacdias/micropub-parser')
const indieauth = require('@hacdias/indieauth-middleware')
const transformer = require('./transformer')

// https://www.w3.org/TR/micropub

const badRequest = (res, reason, code = 400) => {
  debug('invalid request, code: %d; reason: %s', code, reason)
  res.status(code).json({
    error: 'invalid_request',
    error_description: reason
  })
}

const config = Object.freeze({
  'media-endpoint': 'https://api.hacdias.com/micropub',
  'syndicate-to': [
    {
      uid: 'twitter',
      name: 'Twitter'
    }
  ]
})

module.exports = ({ cdn, domain, xray, webmentions, posse, hugo, git, notify, queue, tokenReference }) => {
  const getPhotos = async (post, { meta, content }) => {
    try {
      const newPhotos = await helpers.getPhotos(meta, cdn)

      if (newPhotos) {
        meta.properties.photo = newPhotos
        await hugo.saveEntry(post, { meta, content })
        await git.commit(`cdn photos on ${post}`)
      }
    } catch (e) {
      debug('could not update post %s', post)
    }
  }

  const sendWebmentions = async (post, url, target) => {
    const targets = []

    if (target) {
      targets.push(target)
    }

    try {
      const html = await hugo.getEntryHTML(post)
      const mentions = await helpers.getMentions(url, html)
      targets.push(...mentions)
    } catch (err) {
      notify.sendError(err)
    }

    try {
      await webmentions.send({ source: url, targets })
    } catch (err) {
      notify.sendError(err)
    }
  }

  const receive = async (req, res, data) => {
    const { meta, content, slug, type, relatedURL } = transformer.createPost(data)

    if (relatedURL) {
      try {
        await xray.requestAndSave(relatedURL)
      } catch (e) {
        notify.sendError(e)
      }
    }

    const { post } = await hugo.newEntry({ meta, content, slug })
    const url = `${domain}${post}`

    res.redirect(202, url)

    await git.commit(`add ${post}`)
    hugo.build()

    notify.send(`📄 Post published: ${url}`)

    // This can be processed async.
    sendWebmentions(post, url, relatedURL)

    await getPhotos(post, { meta, content })

    const syndication = await posse({
      content,
      url,
      type,
      commands: data.commands,
      relatedURL
    })

    if (syndication.length === 0) {
      return
    }

    try {
      const { meta, content } = await hugo.getEntry(post)
      meta.properties = meta.properties || {}
      meta.properties.syndication = syndication
      await hugo.saveEntry(post, { meta, content })
      await git.commit(`syndication on ${post}`)
    } catch (err) {
      debug('could not save syndication %s', err.stack)
      notify.sendError(err)
    }
  }

  const source = async (url) => {
    if (!url.startsWith(domain)) {
      throw new Error('invalid request')
    }

    const post = url.replace(domain, '', 1)
    const { meta, content } = await hugo.getEntry(post)

    const entry = {
      type: ['h-entry'],
      properties: meta.properties
    }

    if (meta.title) {
      entry.properties.name = [meta.title]
    }

    if (meta.tags) {
      entry.properties.category = meta.tags
    }

    if (content) {
      entry.properties.content = [content]
    }

    if (meta.date) {
      entry.properties.published = [meta.date]
    }

    return entry
  }

  const update = async (req, res, data) => {
    const post = data.url.replace(domain, '', 1)
    const entry = transformer.updatePost(await hugo.getEntry(post), data)

    await hugo.saveEntry(post, entry)
    await git.commit(`update ${post}`)

    res.redirect(200, data.url)
    queue.add(() => getPhotos(post, entry))
  }

  const remove = async (req, res, data) => {
    const post = data.url.replace(domain, '', 1)
    const { meta, content } = await hugo.getEntry(post)

    meta.expiryDate = new Date()
    await hugo.saveEntry(post, { meta, content })
    await git.commit(`delete ${post}`)

    res.sendStatus(200)
  }

  const unremove = async (req, res, data) => {
    const post = data.url.replace(domain, '', 1)
    const entry = await hugo.getEntry(post)

    if (!entry.meta.expiryDate) {
      return res.sendStatus(400)
    }

    delete entry.meta.expiryDate

    await hugo.saveEntry(post, entry)
    await git.commit(`delete ${post}`)
  }

  const media = async (req, res) => {
    debug('media file received')
    const hash = sha256(req.file.buffer)
    const ext = extname(
      req.file.originalname ||
        '.' + mime.extension(req.file.mimetype)
    )

    const filename = `${hash}${ext}`
    const url = await cdn.upload(req.file.buffer, filename)

    debug('media file uploaded to %s', url)
    return res.redirect(201, url)
  }

  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())
  router.use(express.urlencoded({ extended: true }))
  router.use(indieauth(tokenReference))

  const storage = multer.memoryStorage()
  const upload = multer({ storage: storage })

  router.use(upload.single('file'))

  router.get('/', ar(async (req, res) => {
    debug('GET received; query: %o', req.query)

    switch (req.query.q) {
      case 'source':
        if (typeof req.query.url !== 'string') {
          return badRequest(res, 'url must be set on source query')
        }

        return res.json(await queue.add(() => source(req.query.url)))
      case 'config':
        return res.json(config)
      case 'syndicate-to':
        return res.json({ 'syndicate-to': config['syndicate-to'] })
      default:
        return badRequest(res, 'invalid query')
    }
  }))

  router.post('/', ar(async (req, res) => {
    debug('POST received')
    let request

    if (req.file) {
      return media(req, res)
    }

    try {
      if (req.is('json')) {
        request = parseJson(req.body)
      } else {
        request = parseFormEncoded(req.body)
      }
    } catch (e) {
      return badRequest(res, e.stack)
    }

    debug('POST transformed')

    switch (request.action) {
      case 'create':
        await queue.add(() => receive(req, res, request))
        break
      case 'update':
        await queue.add(() => update(req, res, request))
        break
      case 'delete':
        await queue.add(() => remove(req, res, request))
        break
      case 'undelete':
        await queue.add(() => unremove(req, res, request))
        break
      default:
        return badRequest(res, 'invalid request')
    }

    try {
      if (request.action === 'delete') {
        hugo.buildAndClean()
      } else {
        hugo.build()
      }
    } catch (err) {
      debug('could not rebuild website %s', err.stack)
      notify.sendError(err)
    }
  }))

  return router
}
