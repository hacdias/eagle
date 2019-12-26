const express = require('express')
const { badRequest } = require('./utils')
const { parseJson, parseFormEncoded } = require('./body-parser')

module.exports = (options) => {
  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())
  router.use(express.urlencoded({ extended: true }))

  router.use((req, res, next) => {
    if (req.body) {
      if (req.is('json')) {
        req.body = parseJson(req.body)
      } else {
        req.body = parseFormEncoded(req.body)
      }
    }

    next()
  })

  router.use((req, res, next) => {
    if (/* TODO: invalid token */ false) {
      res.status(403).json({
        error: 'forbidden'
      })

      return
    }

    next()
  })

  router.get('/', (req, res) => {
    console.log(res.body)
    res.sendStatus(200)
  })

  router.post('/', (req, res) => {
    if (req.query.q) {
      return badRequest(res, 'Queries only supported with GET method', 405)
    } else if (req.body.mp && req.body.mp.action) {
      // TODO
      return badRequest(res, 'This endpoint does not yet support updates.', 501)
    } else if (!req.body.type) {
      return badRequest(res, 'Missing "h" value.')
    }

    console.log(res.body)
    res.sendStatus(200)
  })

  return router
}
