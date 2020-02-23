const { ar } = require('./utils')
const crypto = require('crypto')

module.exports = ({ git, hugo, secret, queue }) => ar(async (req, res) => {
  const sig = 'sha1=' + crypto
    .createHmac('sha1', secret)
    .update(JSON.stringify(req.body))
    .digest('hex')

  if (req.headers['x-hub-signature'] !== sig) {
    return res.sendStatus(403)
  }

  await queue.add(async () => {
    await git.pull()
    await git.push()
    await hugo.build()
  })

  res.sendStatus(200)
})
