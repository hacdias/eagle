const { join } = require('path')
const fs = require('fs-extra')
const yaml = require('js-yaml')
const { run } = require('./utils')

module.exports = function createHugo ({ dir, publicDir }) {
  const contentDir = join(dir, 'content')
  const dataDir = join(dir, 'data')

  const build = () => run('hugo', ['--minify', '--destination', publicDir], {
    cwd: dir
  })

  const buildAndClean = () => run('hugo', ['--minify', '--gc', '--cleanDestinationDir', '--destination', publicDir], {
    cwd: dir
  })

  // Creates a new entry from metadata, content and a slug and returns
  // an object { post, path } where post is the post directory as in uri
  // and path is the local path in disk.
  const newEntry = async ({ meta, content, slug }) => {
    const year = meta.date.getFullYear().toString()
    const month = (meta.date.getMonth() + 1).toString().padStart(2, '0')
    const day = meta.date.getDate().toString().padStart(2, '0')

    const num = getNextPostNumber(year, month, day)
    let path = `/${year}/${month}/${day}/${num}/`

    if (slug !== '') {
      meta.aliases = [path]
      path += `${slug}/`
    }

    return saveEntry(path, { meta, content })
  }

  // Saves an already existing entry. Takes a post path (uri) and an object
  // with { meta, content }.
  const saveEntry = async (post, { meta, content }) => {
    const path = join(contentDir, post)
    const index = join(path, 'index.md')
    const val = `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content}`
    await fs.outputFile(index, val)
    return { post, path }
  }

  // Gets an entry as a { meta, content } object.
  const getEntry = async (post) => {
    const index = join(contentDir, post, 'index.md')
    const file = (await fs.readFile(index)).toString()
    const [frontmatter, content] = file.split('\n---')

    return {
      meta: yaml.safeLoad(frontmatter),
      content: content.trim()
    }
  }

  const getEntryHTML = async (post) => {
    const index = join(publicDir, post, 'index.html')
    return (await fs.readFile(index)).toString()
  }

  const getNextPostNumber = (year, month, day) => {
    const pathToCheck = join(contentDir, year, month, day)
    fs.ensureDirSync(pathToCheck)

    const lastNum = Math.max(
      ...fs.readdirSync(pathToCheck)
        .filter(f => fs.statSync(join(pathToCheck, f)).isDirectory())
        .map(n => parseInt(n)),
      0
    )

    return (lastNum + 1).toString()
  }

  return Object.freeze({
    contentDir,
    dataDir,
    publicDir,
    dir,

    build,
    buildAndClean,
    newEntry,
    saveEntry,
    getEntry,
    getEntryHTML
  })
}
