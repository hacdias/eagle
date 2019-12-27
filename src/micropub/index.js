const express = require('express')
const got = require('got')
const debug = require('debug')('micropub')
const multer = require('multer')
const { parseJson, parseFormEncoded } = require('./body-parser')

// https://www.w3.org/TR/micropub

const requiredScopes = Object.freeze([
  'create',
  'update',
  'delete'
])

const badRequest = (res, reason, code = 400) => {
  debug('invalid request, code: %d; reason: %s', code, reason)
  res.status(code).json({
    error: 'invalid_request',
    error_description: reason
  })
}

const getAuth = async (token, endpoint) => {
  debug('getting token info from %s', endpoint)

  const { body } = await got(endpoint, {
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`
    },
    responseType: 'json'
  })

  return body
}

module.exports = ({ queryHandler, postHandler, mediaHandler, tokenReference }) => {
  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())
  router.use(express.urlencoded({ extended: true }))

  const storage = multer.memoryStorage()
  const upload = multer({ storage: storage })

  router.use(upload.single('file'))

  router.use((req, res, next) => {
    let token

    if (req.headers.authorization) {
      token = req.headers.authorization.trim().split(/\s+/)[1]
    } else if (!token && req.body && req.body.access_token) {
      token = req.body.access_token
    }

    if (!token) {
      debug('missing authentication token')
      return res.status(401).json({
        error: 'unauthorized',
        error_description: 'missing authentication token'
      })
    }

    getAuth(token, tokenReference.endpoint)
      .then(data => {
        if (!data.me || !data.scope || Array.isArray(data.me) || Array.isArray(data.scope)) {
          debug('invalid response from endpoint')
          return res.status(403).json({
            error: 'forbidden',
            error_description: 'invalid response from endpoint'
          })
        }

        if (data.me !== tokenReference.me) {
          debug('user is not allowed')
          return res.status(403).json({
            error: 'forbidden',
            error_description: 'user not allowed'
          })
        }

        const scopes = data.scope.split(' ')

        for (const scope of requiredScopes) {
          if (!scopes.includes(scope)) {
            debug('user does not have required scopes: %o, has %o', requiredScopes, scopes)
            return res.status(401).json({
              error: 'insufficient_scope',
              error_description: `requires scopes: ${requiredScopes.join(', ')}`
            })
          }
        }

        next()
      })
      .catch(e => {
        debug('internal error on auth: %s', e.toString())
        res.status(500).json({
          error: 'internal server error'
        })
      })
  })

  router.get('/', (req, res) => {
    debug('received GET request with query %o', req.query)

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
        debug('internal error on query handler: %s', e.toString())
        res.status(500).json({
          error: 'internal server error'
        })
      })
  })

  router.post('/', (req, res) => {
    debug('received POST request')
    let request

    if (req.file) {
      mediaHandler(req.file)
        .then(loc => res.redirect(201, loc))
        .catch(e => {
          debug('internal error on media handler: %s', e.toString())
          res.status(500).json({
            error: 'internal server error'
          })
        })

      return
    }

    try {
      if (req.is('json')) {
        request = parseJson(req.body)
      } else {
        request = parseFormEncoded(req.body)
      }
    } catch (e) {
      return badRequest(res, e.toString())
    }

    postHandler(request)
      .then(loc => res.redirect(201, loc))
      .catch(e => {
        debug('internal error on post handler: %s', e.toString())
        res.status(500).json({
          error: 'internal server error'
        })
      })
  })

  return router
}
