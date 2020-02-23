const { ar } = require('./utils')
const crypto = require('crypto')
const execa = require('execa')

module.exports = ({ git, notesRepo, buildKB, hugo, secret, queue }) => ar(async (req, res) => {
  console.log(req.body)
  const sig = 'sha1=' + crypto
    .createHmac('sha1', secret)
    .update(JSON.stringify(req.body))
    .digest('hex')

  if (req.headers['x-hub-signature'] !== sig) {
    return res.sendStatus(403)
  }

  await queue.add(async () => {
    await execa('git', ['pull'], { cwd: notesRepo })
    await buildKB()
    await git.commit('update kb')
    await hugo.build()
  })

  res.sendStatus(200)
})
