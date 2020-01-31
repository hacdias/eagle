const { spawnSync } = require('child_process')
const crypto = require('crypto')

function sha256 (data) {
  return crypto.createHash('sha256').update(data).digest('hex')
}

function run () {
  const res = spawnSync(...arguments)
  if (res.error) throw res.error
}

module.exports = {
  run,
  sha256
}
