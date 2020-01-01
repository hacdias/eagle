const crypto = require('crypto')
const { spawnSync } = require('child_process')

function sha256 (data) {
  return crypto.createHash('sha256').update(data).digest('hex')
}´

function run () {
  const res = spawnSync(...arguments)
  const stderr = res.stderr.toString()
  if (stderr.length) throw new Error(stderr)
  if (res.error) throw res.error
}

module.exports = {
  sha256,
  run
}
