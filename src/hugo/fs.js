const { join } = require('path')
const fs = require('fs-extra')
const yaml = require('js-yaml')

const getNextPostNumber = (contentDir, year, month, day) => {
  const pathToCheck = join(contentDir, year, month, day)
  fs.ensureDirSync(pathToCheck)

  const lastNum = fs.readdirSync(pathToCheck)
    .filter(f => fs.statSync(join(pathToCheck, f)).isDirectory())
    .sort().pop() || '00'

  return (parseInt(lastNum) + 1).toString().padStart(2, '0')
}

const makePost = ({ date, slug, meta, content, contentDir }) => {
  const year = date.getFullYear().toString()
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')

  const num = getNextPostNumber(contentDir, year, month, day)
  const alias = `/${year}/${month}/${day}/${num}/`
  const path = `${alias}${slug}`

  if (slug !== '') {
    meta.aliases = [alias]
  }

  const dirPath = join(contentDir, path)
  const indexPath = join(dirPath, 'index.md')
  const index = `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content}`

  fs.ensureDirSync(dirPath, { recursive: true })
  fs.writeFileSync(indexPath, index)

  return path
}

module.exports = {
  makePost
}
