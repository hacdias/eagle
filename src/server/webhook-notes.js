const debug = require('debug')('eagle:server:hook:notes')
const crypto = require('crypto')
const execa = require('execa')
const { join } = require('path')
const ar = require('../utils/ar')
const buildKB = require('../builders/kb')

module.exports = ({ services, repositoryDir, secret }) => ar(async (req, res) => {
  const { hugo, git, queue } = services

  const sig = 'sha1=' + crypto
    .createHmac('sha1', secret)
    .update(JSON.stringify(req.body))
    .digest('hex')

  if (req.headers['x-hub-signature'] !== sig) {
    return res.sendStatus(403)
  }

  res.sendStatus(202)

  const src = join(repositoryDir, 'notes')
  const dst = join(hugo.dir, 'content', 'kb')
  debug('building from %s: %s', src, dst)

  await queue.add(async () => {
    debug('git pulling notes repo')
    await execa('git', ['pull'], { cwd: repositoryDir })
    await buildKB({ src, dst })
    await git.commit('update kb')
    await hugo.build()
  })
})
