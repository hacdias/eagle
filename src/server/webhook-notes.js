const ar = require('../utils/ar')
const crypto = require('crypto')

const createGit = require('../services/git')
const createHugo = require('../services/hugo')

module.exports = ({ secret, src, dst }) => {
  const git = createGit({ cwd: src })
  const hugo = createHugo({ dir: src, publicDir: dst })

  return ar(async (req, res) => {
    const sig = 'sha1=' + crypto
      .createHmac('sha1', secret)
      .update(JSON.stringify(req.body))
      .digest('hex')

    if (req.headers['x-hub-signature'] !== sig) {
      return res.sendStatus(403)
    }

    await git.pull()
    await hugo.buildAndClean()
    res.sendStatus(200)
  })
}
