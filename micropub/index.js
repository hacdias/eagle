const express = require('express')
const { badRequest } = require('./utils')
const { parseJson, parseFormEncoded } = require('./body-parser')

module.exports = ({ postHandler }) => {
  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())
  router.use(express.urlencoded({ extended: true }))

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
      console.log(e)
      res.status(500).json({
        error: 'internal server error'
      })
    }
  })

  return router
}
