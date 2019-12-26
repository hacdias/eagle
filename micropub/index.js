const express = require('express')
const { badRequest } = require('./utils')
const { parseJson, parseFormEncoded, processFiles } = require('./body-parser')

module.exports = ({ postHandler }) => {
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

    if (req.files && Object.getOwnPropertyNames(req.files)[0]) {
      req.body = processFiles(req.body, req.files)
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

    try {
      const location = postHandler(req.body)
      res.redirect(location, 201)
    } catch (e) {
      console.log(e)
      res.status(500).json({
        error: 'internal server error'
      })
    }
  })

  return router
}
