const { run } = require('./utils')

module.exports = function createGit (opts) {
  const commit = (message) => {
    run('git', ['add', '-A'], opts)
    run('git', ['commit', '-m', message], opts)
  }

  const push = () => {
    run('git', ['push'], opts)
  }

  return Object.freeze({
    commit,
    push
  })
}
