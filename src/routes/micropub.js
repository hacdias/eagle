const express = require('express')
const debug = require('debug')('micropub')
const multer = require('multer')
const { ar } = require('./utils')

const { parseJson, parseFormEncoded } = require('@hacdias/micropub-parser')
const indieauth = require('@hacdias/indieauth-middleware')

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

module.exports = ({ eagle, tokenReference }) => {
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
    debug('received GET request with query %o', req.query)

    switch (req.query.q) {
      case 'source':
        if (typeof req.query.url !== 'string') {
          return badRequest(res, 'url must be set on source query')
        }

        return res.json(await eagle.sourceMicropub(req.query.url))
      case 'config':
        return res.json(config)
      case 'syndicate-to':
        return res.json({ 'syndicate-to': config['syndicate-to'] })
      default:
        return badRequest(res, 'invalid query')
    }
  }))

  router.post('/', ar(async (req, res) => {
    debug('received POST request')
    let request

    if (req.file) {
      debug('media handler not implemented')
      return res.sendStatus(501)
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

    switch (request.action) {
      case 'create':
        return eagle.receiveMicropub(req, res, request)
      case 'update':
        return eagle.updateMicropub(req, res, request)
      case 'delete':
        return eagle.deleteMicropub(req, res, request)
      case 'undelete':
        return eagle.undeleteMicropub(req, res, request)
      default:
        throw new Error('invalid request')
    }
  }))

  return router
}
