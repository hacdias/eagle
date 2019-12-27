const fs = require('fs')
const yaml = require('js-yaml')

module.exports = class HugoManager {
  newPost ({ properties, commands }) {
    const date = new Date()
    const year = date.getFullYear().toString()
    const month = (date.getMonth() + 1).toString().padStart(2, '0')
    const day = date.getDate().toString().padStart(2, '0')

    const meta = {
      title: null,
      description: null,
      date,
      categories: [],
      tags: [],
      aliases: [
        `/${year}/${month}/${day}/TODO/`
      ]
    }

    console.log(yaml.safeDump(meta))

    const content = properties.content
      ? properties.content.join('\n')
      : ''

    const file = `---\n${yaml.safeDump(meta, { sortKeys: true })}\n---\n${content}`
    console.log(file)

    const path = `/${year}/${month}/${day}/TODO/`

    return `https://hacdias.com${path}`
  }
}
