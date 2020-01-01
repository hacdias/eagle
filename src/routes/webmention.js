const express = require('express')
const debug = require('debug')('routes:webmention')

module.exports = ({ eagle, secret }) => {
  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())

  router.post('/', (req, res) => {
    debug('incoming webmention')

    if (req.body.secret !== secret) {
      debug('invalid secret')
      return res.sendStatus(403)
    }

    delete req.body.secret

    eagle.receiveWebMention(req.body)
      .then(() => {
        debug('webmention handled')
        res.sendStatus(200)
      })
      .catch(e => {
        debug('error while handling webmention %s', e.stack)
        res.sendStatus(500)
      })
  })

  return router
}
