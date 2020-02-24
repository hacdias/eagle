const { ar } = require('./utils')
const { join } = require('path')
const debug = require('debug')('eagle:server:build:watches')
const buildWatches = require('../build-watches')

module.exports = ({ git, secret, queue, source, hugo }) => ar(async (req, res) => {
  if (req.query.secret !== secret) {
    return res.sendStatus(403)
  }

  res.sendStatus(202)

  await queue.add(async () => {
    const relativePath = 'data/watches.json'
    const dst = join(hugo.dir, relativePath)
    debug('building from %s to %s', source, dst)

    await buildWatches({ src: source, dst })

    const { stdout } = await git.diff(relativePath)
    if (stdout === '') {
      debug('%s was not changed, skipping commit', dst)
      // No updates were made.
      return
    }

    const { meta, content } = await hugo.getEntry('/watches')
    meta.date = new Date()
    await hugo.saveEntry('/watches', { meta, content })

    await git.commitFile([relativePath, 'content/watches/index.md'], 'update watches')
    await hugo.build()
    debug('committed and built')
  })
})
