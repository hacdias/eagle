const { join } = require('path')
const execa = require('execa')
const fs = require('fs-extra')
const yaml = require('js-yaml')
const converter = require('./converter')

module.exports = function createHugo ({ dir, publicDir }) {
  const contentDir = join(dir, 'content')
  const dataDir = join(dir, 'data')

  const build = () => execa.sync('hugo', ['--minify', '--destination', publicDir], {
    cwd: dir
  })

  const buildAndClean = () => execa.sync('hugo', ['--minify', '--gc', '--cleanDestinationDir', '--destination', publicDir], {
    cwd: dir
  })

  // Creates a new entry from metadata, content and a slug and returns
  // an object { post, path } where post is the post directory as in uri
  // and path is the local path in disk.
  const newEntry = async ({ meta, content, slug, type }, opts) => {
    type = type === 'article' ? 'article' : 'micro'
    const year = meta.date.getFullYear().toString()
    const month = (meta.date.getMonth() + 1).toString().padStart(2, '0')
    const path = `/${type}/${year}/${month}/${slug}/`
    return saveEntry(path, { meta, content }, opts)
  }

  // Saves an already existing entry. Takes a post path (uri) and an object
  // with { meta, content }.
  const saveEntry = async (post, { meta, content }, { keepOriginal = false } = {}) => {
    if (meta.properties && !keepOriginal) {
      meta.properties = converter.mf2ToInternal(meta.properties)
    }

    const path = join(contentDir, post)
    const index = join(path, 'index.md')
    const val = `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content}`
    await fs.outputFile(index, val)
    return { post, path }
  }

  // Gets an entry as a { meta, content } object.
  const getEntry = async (post, { keepOriginal = false } = {}) => {
    const index = join(contentDir, post, 'index.md')
    const file = (await fs.readFile(index)).toString()
    const [frontmatter] = file.split('\n---', 2)
    const content = file.replace(frontmatter + '\n---', '').trim()
    const meta = yaml.safeLoad(frontmatter)
    meta.properties = meta.properties || {}

    if (meta.properties && !keepOriginal) {
      meta.properties = converter.internalToMf2(meta.properties)
    }

    return {
      post,
      meta,
      content: content.trim()
    }
  }

  const getEntryHTML = async (post) => {
    const index = join(publicDir, post, 'index.html')
    return (await fs.readFile(index)).toString()
  }

  const getAll = async function * (type, opts) {
    let dir = contentDir
    if (type) dir = join(dir, type)
    const files = getAllFiles(dir)
      .filter(p => p.endsWith('/index.md'))
      .map(p => {
        p = p.replace('/index.md', '', 1)
        p = p.replace(contentDir, '', 1)
        return p
      })

    for (const file of files) {
      yield getEntry(file, opts)
    }
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
    getAll,
    getEntry,
    getEntryHTML
  })
}

const getAllFiles = function (dirPath, arrayOfFiles) {
  const files = fs.readdirSync(dirPath)

  arrayOfFiles = arrayOfFiles || []

  files.forEach(function (file) {
    if (fs.statSync(dirPath + '/' + file).isDirectory()) {
      arrayOfFiles = getAllFiles(dirPath + '/' + file, arrayOfFiles)
    } else {
      arrayOfFiles.push(join(dirPath, '/', file))
    }
  })

  return arrayOfFiles
}
