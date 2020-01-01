const express = require('express')
const debug = require('debug')('micropub')
const multer = require('multer')

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

module.exports = ({ queryHandler, postHandler, mediaHandler, tokenReference }) => {
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

  router.get('/', (req, res) => {
    debug('received GET request with query %o', req.query)

    if (!queryHandler) {
      debug('query handler not implemented')
      return res.sendStatus(501)
    }

    switch (req.query.q) {
      case 'source':
        if (typeof req.query.url !== 'string') {
          return badRequest(res, 'url must be set on source query')
        }

        break
      case 'config':
      case 'syndicate-to':
        break
      default:
        return badRequest(res, 'invalid query')
    }

    queryHandler(req.query)
      .then(j => res.json(j))
      .catch(e => {
        debug('internal error on query handler: %s', e.stack)
        res.status(500).json({
          error: 'internal server error'
        })
      })
  })

  router.post('/', (req, res) => {
    debug('received POST request')
    let request

    if (req.file) {
      if (!mediaHandler) {
        debug('media handler not implemented')
        return res.sendStatus(501)
      }

      if (!req.hasScope(['media'])) {
        return
      }

      mediaHandler(req.file)
        .then(loc => res.redirect(201, loc))
        .catch(e => {
          debug('internal error on media handler: %s', e.stack)
          res.status(500).json({
            error: 'internal server error'
          })
        })

      return
    }

    if (!postHandler) {
      debug('post handler not implemented')
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

    postHandler(request, req.hostname)
      .then(loc => res.redirect(201, loc))
      .catch(e => {
        debug('internal error on post handler: %s', e.stack)
        res.status(500).json({
          error: 'internal server error'
        })
      })
  })

  return router
}
