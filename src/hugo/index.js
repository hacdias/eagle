const fs = require('fs-extra')
const path = require('path')
const yaml = require('js-yaml')

module.exports = class HugoManager {
  constructor ({ contentDir }) {
    this.contentDir = contentDir
  }

  _getNextPostNumber (year, month, day) {
    const pathToCheck = path.join(this.contentDir, year, month, day)
    fs.ensureDirSync(pathToCheck)

    const lastNum = fs.readdirSync(pathToCheck)
      .filter(f => fs.statSync(f).isDirectory())
      .sort().pop() || '00'

    return (parseInt(lastNum) + 1).toString().padStart(2, '0')
  }

  newPost ({ properties, commands }) {
    const date = new Date()
    const year = date.getFullYear().toString()
    const month = (date.getMonth() + 1).toString().padStart(2, '0')
    const day = date.getDate().toString().padStart(2, '0')

    const content = properties.content
      ? properties.content.join('\n').trim()
      : ''

    delete properties.content

    const meta = {
      title: properties.name
        ? properties.name.join(' ').trim()
        : '',
      description: null,
      date,
      categories: [],
      tags: [],
      aliases: [],
      properties: {}
    }

    delete properties.name

    if (meta.title === '' && content === '') {
      throw new Error('must have title or content')
    }

    if (properties.category) meta.tags = properties.category

    const slug = commands['mp-slug']
      ? commands['mp-slug'][0]
      : ''

    meta.properties = properties

    const num = this._getNextPostNumber(year, month, day)
    const alias = `/${year}/${month}/${day}/${num}/`
    const url = `${alias}${slug}`

    const dirPath = path.join(this.contentDir, url)
    const indexPath = path.join(dirPath, 'index.md')
    const index = `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content}`

    fs.ensureDirSync(dirPath, { recursive: true })
    fs.writeFileSync(indexPath, index)

    return `https://hacdias.com${url}`
  }
}
