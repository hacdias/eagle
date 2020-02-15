const debug = require('debug')('eagle:server:webmention')
const { ar } = require('./utils')

module.exports = ({ webmentions, hugo, notify, queue, secret }) => ar(async (req, res) => {
  debug('incoming webmention')

  if (req.body.secret !== secret) {
    debug('invalid secret')
    return res.sendStatus(403)
  }

  delete req.body.secret
  await queue.add(() => webmentions.receive(req.body))
  res.sendStatus(200)

  try {
    hugo.build()
    notify.send(`💬 ${req.body.deleted ? 'Deleted' : 'Received'} webmention: ${req.body.target}`)
  } catch (err) {
    debug('error on post-webmention processor %s', err.stack)
    notify.sendError(err)
  }

  debug('webmention handled')
})
