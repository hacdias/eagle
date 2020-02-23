const { ar } = require('./utils')
const debug = require('debug')('eagle:server:notes')
const crypto = require('crypto')

module.exports = ({ git, buildKB, secret }) => ar(async (req, res) => {
  console.log(req.body)
  const sig = 'sha1=' + crypto.createHmac('sha1', secret).update(req.body.toString()).digest('hex')

  if (req.headers['x-hub-signature'] !== sig) {
    return res.sendStatus(403)
  }

  console.log('should update')
  return res.sendStatus(200)
})
