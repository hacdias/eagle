const execa = require('execa')

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
    await run('git', ['add', '-A'], opts)
    return run('git', ['commit', '-m', message], opts)
  }

  const push = async () => {
    return run('git', ['push'], opts)
  }

  const pull = async () => {
    return run('git', ['pull'], opts)
  }

  return Object.freeze({
    commit,
    push,
    pull
  })
}
