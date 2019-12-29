const { spawnSync } = require('child_process')

const commit = (message, opts) => {
  let res = spawnSync('git', ['add', '-A'], opts)
  if (res.error) throw res.error
  res = spawnSync('git', ['commit', '-m', message], opts)
  if (res.error) throw res.error
}

const push = (opts) => {
  const { error } = spawnSync('git', ['push'], opts)
  if (error) throw error
}

module.exports = {
  commit,
  push,
  configuredGit: (opts) => ({
    commit: msg => commit(msg, opts),
    push: () => push(opts)
  })
}
