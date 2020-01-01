const { join } = require('path')
const fs = require('fs-extra')
const yaml = require('js-yaml')
const { run } = require('./utils')

class HugoService {
  constructor ({ dir, publicDir, domain }) {
    this.dir = dir
    this.contentDir = join(dir, 'content')
    this.dataDir = join(dir, 'data')
    this.publicDir = publicDir
    this.domain = domain
  }

  build () {
    run('hugo', ['--minify', '--destination', this.publicDir], {
      cwd: this.dir
    })
  }

  buildAndClean () {
    run('hugo', ['--minify', '--gc', '--cleanDestinationDir', '--destination', this.publicDir], {
      cwd: this.dir
    })
  }

  // Creates a new entry from metadata, content and a slug and returns
  // an object { post, path } where post is the post directory as in uri
  // and path is the local path in disk.
  async newEntry ({ meta, content, slug }) {
    const year = meta.date.getFullYear().toString()
    const month = (meta.date.getMonth() + 1).toString().padStart(2, '0')
    const day = meta.date.getDate().toString().padStart(2, '0')

    const num = this._getNextPostNumber(this.contentDir, year, month, day)
    let path = `/${year}/${month}/${day}/${num}/`

    if (slug !== '') {
      meta.aliases = [path]
      path += `${slug}/`
    }

    return this.saveEntry(path, { meta, content })
  }

  // Saves an already existing entry. Takes a post path (uri) and an object
  // with { meta, content }.
  async saveEntry (post, { meta, content }) {
    const path = join(this.contentDir, post)
    const index = join(path, 'index.md')
    const val = `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content}`
    await fs.outputFile(index, val)
    return { post, path }
  }

  // Gets an entry as a { meta, content } object.
  async getEntry (post) {
    const index = join(this.contentDir, post, 'index.md')
    const file = (await fs.readFile(index)).toString()
    const [frontmatter, content] = file.split('\n---')

    return {
      meta: yaml.safeLoad(frontmatter),
      content: content.trim()
    }
  }

  async getEntryHTML (post) {
    const index = join(this.publicDir, post, 'index.html')
    return (await fs.readFile(index)).toString()
  }

  _getNextPostNumber (year, month, day) {
    const pathToCheck = join(this.contentDir, year, month, day)
    fs.ensureDirSync(pathToCheck)

    const lastNum = fs.readdirSync(pathToCheck)
      .filter(f => fs.statSync(join(pathToCheck, f)).isDirectory())
      .sort().pop() || '00'

    return (parseInt(lastNum) + 1).toString().padStart(2, '0')
  }
}

module.exports = HugoService
