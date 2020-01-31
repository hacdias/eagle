const debug = require('debug')('eagle:server:webmention')
const { ar } = require('./utils')

module.exports = ({ webmentions, hugo, telegram, queue, secret }) => ar(async (req, res) => {
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
    telegram.send(`💬 Received webmention: ${req.body.target}`)
  } catch (e) {
    // TODO:
    debug('error on post-webmention processor %s', e.stack)
  }

  debug('webmention handled')
})
