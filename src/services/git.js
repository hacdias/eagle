const { run } = require('./utils')

module.exports = function createGit (opts) {
  const commit = (message) => {
    run('git', ['add', '-A'], opts)
    run('git', ['commit', '-m', message], opts)
  }

  const commitFile = (message, ...files) => {
    run('git', ['commit', '-m', message, '-o', ...files], opts)
  }

  const push = () => {
    run('git', ['push'], opts)
  }

  const pull = () => {
    run('git', ['pull'], opts)
  }

  return Object.freeze({
    commit,
    commitFile,
    push,
    pull
  })
}
