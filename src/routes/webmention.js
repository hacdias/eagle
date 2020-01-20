const express = require('express')
const debug = require('debug')('routes:webmention')
const { ar } = require('./utils')

module.exports = ({ eagle, secret }) => {
  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())

  router.post('/', ar(async (req, res) => {
    debug('incoming webmention')

    if (req.body.secret !== secret) {
      debug('invalid secret')
      return res.sendStatus(403)
    }

    delete req.body.secret

    try {
      await eagle.receiveWebmention(req.body)
      res.sendStatus(200)
      debug('webmention handled')
    } catch (e) {
      debug('error while handling webmention %s', e.stack)
      res.sendStatus(500)
    }
  }))

  return router
}
