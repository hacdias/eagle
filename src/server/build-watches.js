const debug = require('debug')('eagle:server:build:watches')
const { join } = require('path')
const buildWatches = require('../builders/watches')
const ar = require('../utils/ar')

module.exports = ({ services, secret, repositoryDir }) => ar(async (req, res) => {
  if (req.query.secret !== secret) {
    return res.sendStatus(403)
  }

  res.sendStatus(202)

  const { queue, hugo, git } = services

  await queue.add(async () => {
    const relativePath = 'data/watches.json'
    const dst = join(hugo.dir, relativePath)
    debug('building from %s to %s', repositoryDir, dst)

    await buildWatches({ src: repositoryDir, dst })

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

    // Take advantage of this being updated once a day to push at least once
    // a day too.
    await git.push()
  })
})
