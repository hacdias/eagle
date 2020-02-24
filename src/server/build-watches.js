const { ar } = require('./utils')
const { join } = require('path')
const debug = require('debug')('eagle:server:build:watches')
const buildWatches = require('../build-watches')

module.exports = ({ git, secret, queue, source, hugo }) => ar(async (req, res) => {
  if (req.query.secret !== secret) {
    return res.sendStatus(403)
  }

  await queue.add(async () => {
    const relativePath = 'data/watches.json'
    const output = join(hugo.dir, relativePath)
    debug('building from %s to %s', source, output)

    await buildWatches({ source, output })

    const { stdout } = await git.diff(relativePath)
    if (stdout === '') {
      debug('%s was not changed, skipping commit', output)
      // No updates were made.
      return
    }

    await git.commitFile(relativePath, 'update watches')
    await hugo.build()
    debug('committed and built')
  })

  return res.sendStatus(200)
})
