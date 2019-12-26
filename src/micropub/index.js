const express = require('express')
const got = require('got')
const { badRequest } = require('./utils')
const { parseJson, parseFormEncoded } = require('./body-parser')

const authenticate = async (token, endpoint, me) => {
  const { body } = await got(endpoint, {
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`
    },
    responseType: 'json'
  })

  if (!body.me || !body.scope || Array.isArray(body.me) || Array.isArray(body.scope)) {
    throw new Error('invalid token')
  }

  if (body.me !== me) {
    throw new Error('forbidden')
  }

  // TODO: check for multiple users
  // TODO: check for scopes
}

module.exports = ({ postHandler, tokenReference }) => {
  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())
  router.use(express.urlencoded({ extended: true }))

  router.use((req, res, next) => {
    let token

    if (req.headers.authorization) {
      token = req.headers.authorization.trim().split(/\s+/)[1]
    } else if (!token && req.body && req.body.access_token) {
      token = req.body.access_token
    }

    if (!token) {
      return badRequest(res, 'missing "Authorization" header or body parameter.', 401)
    }

    authenticate(token, tokenReference.endpoint, tokenReference.me)
      .then(next)
      .catch(e => {
        res.status(403).json({
          error: 'forbidden'
        })
      })
  })

  router.get('/', (req, res) => {
    switch (req.query.q) {
      case 'config':
        return res.json({})
      case 'source':
        return res.json({})
      case 'syndicate-to':
        return res.json({})
      default:
        badRequest(res, 'invalid query')
    }
  })

  router.post('/', (req, res) => {
    let request

    try {
      if (req.is('json')) {
        request = parseJson(req.body)
      } else {
        request = parseFormEncoded(req.body)
      }
    } catch (e) {
      return badRequest(res, e.toString())
    }

    try {
      const location = postHandler(request)
      res.redirect(201, location)
    } catch (e) {
      res.status(500).json({
        error: 'internal server error'
      })
    }
  })

  return router
}
