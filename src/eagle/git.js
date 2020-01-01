const { run } = require('./utils')

module.exports = class GitService {
  constructor (opts) {
    this.opts = opts
  }

  commit (message) {
    run('git', ['add', '-A'], this.opts)
    run('git', ['commit', '-m', message], this.opts)
  }

  push () {
    run('git', ['push'], this.opts)
  }
}
