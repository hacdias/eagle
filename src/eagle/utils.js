const crypto = require('crypto')
const { spawnSync } = require('child_process')

const sha256 = (data) => crypto.createHash('sha256').update(data).digest('hex')

const run = (args) => {
  const res = spawnSync(...args)
  const stderr = res.stderr.toString()
  if (stderr.length) throw new Error(stderr)
  if (res.error) throw res.error
}

module.exports = {
  sha256,
  run
}
