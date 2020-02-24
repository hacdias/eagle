const execa = require('execa')
const debug = require('debug')('eagle:git')

async function run () {
  const subprocess = execa(...arguments)

  setTimeout(() => {
    subprocess.kill('SIGTERM', {
      forceKillAfterTimeout: 5000
    })
  }, 10000)

  return subprocess
}

module.exports = function createGit (opts) {
  const commit = async (message) => {
    debug('adding')
    await run('git', ['add', '-A'], opts)
    debug('committing')
    return run('git', ['commit', '-m', message], opts)
  }

  const commitFile = async (files, message) => {
    debug('committing %s: %s', files, message)
    return run('git', ['commit', '-m', message, '--', ...files], opts)
  }

  const diff = async (file) => {
    debug('diff %s', file)
    return run('git', ['diff', file], opts)
  }

  const push = async () => {
    debug('pushing')
    return run('git', ['push'], opts)
  }

  const pull = async () => {
    debug('pulling')
    return run('git', ['pull'], opts)
  }

  return Object.freeze({
    commit,
    commitFile,
    push,
    pull,
    diff
  })
}
