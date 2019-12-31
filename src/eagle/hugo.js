const { join } = require('path')
const fs = require('fs-extra')
const yaml = require('js-yaml')
const { spawnSync } = require('child_process')

const getNextPostNumber = (contentDir, year, month, day) => {
  const pathToCheck = join(contentDir, year, month, day)
  fs.ensureDirSync(pathToCheck)

  const lastNum = fs.readdirSync(pathToCheck)
    .filter(f => fs.statSync(join(pathToCheck, f)).isDirectory())
    .sort().pop() || '00'

  return (parseInt(lastNum) + 1).toString().padStart(2, '0')
}

const makePost = ({ slug, meta, content }, { contentDir }) => {
  const year = meta.date.getFullYear().toString()
  const month = (meta.date.getMonth() + 1).toString().padStart(2, '0')
  const day = meta.date.getDate().toString().padStart(2, '0')

  const num = getNextPostNumber(contentDir, year, month, day)
  let path = `/${year}/${month}/${day}/${num}/`

  if (slug !== '') {
    meta.aliases = [path]
    path += `${slug}/`
  }

  const dirPath = join(contentDir, path)
  const indexPath = join(dirPath, 'index.md')
  const index = `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content}`

  fs.ensureDirSync(dirPath, { recursive: true })
  fs.writeFileSync(indexPath, index)

  return path
}

const build = ({ dir, publicDir }) => {
  const res = spawnSync('hugo', [
    '--minify',
    '--destination',
    publicDir
  ], { cwd: dir })

  const stderr = res.stderr.toString()
  if (stderr.length) throw new Error(stderr)
  if (res.error) throw res.error
}

module.exports = {
  makePost,
  build,
  configuredHugo: (opts) => {
    opts.contentDir = join(opts.dir, 'content')

    return {
      makePost: (args) => makePost(args, opts),
      build: () => build(opts)
    }
  }
}
