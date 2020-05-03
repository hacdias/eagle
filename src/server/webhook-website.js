const ar = require('../utils/ar')
const crypto = require('crypto')

module.exports = ({ services, secret }) => ar(async (req, res) => {
  const { queue, git, hugo } = services

  const sig = 'sha1=' + crypto
    .createHmac('sha1', secret)
    .update(JSON.stringify(req.body))
    .digest('hex')

  if (req.headers['x-hub-signature'] !== sig) {
    return res.sendStatus(403)
  }

  await queue.add(async () => {
    await git.pull()
    await hugo.build()
  })

  res.sendStatus(200)
})
