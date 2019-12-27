const express = require('express')
const debug = require('debug')('webmention')

module.exports = ({ hugo }) => {
  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())
  router.use(express.urlencoded({ extended: true }))

  router.post('/', (req, res) => {
    debug('incoming webmention')
    // TODO: check for secret

    delete req.body.secret

    hugo.handleWebMention(req.body)
      .then(() => {
        debug('webmention handled')
        req.status(200)
      })
      .catch(e => {
        debug('error while handling webmention %s', e.toString())
        req.status(500)
      })
  })

  return router
}
