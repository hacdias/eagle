const { spawnSync } = require('child_process')

function run () {
  const res = spawnSync(...arguments)
  const stderr = res.stderr.toString()
  if (stderr.length) throw new Error(stderr)
  if (res.error) throw res.error
}

module.exports = {
  run
}
