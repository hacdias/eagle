const { spawnSync } = require('child_process')

function run () {
  const res = spawnSync(...arguments)
  if (res.error) throw res.error
}

module.exports = {
  run
}
