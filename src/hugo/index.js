const fs = require('fs-extra')
const path = require('path')
const yaml = require('js-yaml')
const slugify = require('@sindresorhus/slugify')
const { spawnSync } = require('child_process')
const pLimit = require('p-limit')
module.exports = class HugoManager {
  constructor ({ dir }) {
    this.limit = pLimit(1)
    this.dir = dir
    this.contentDir = path.join(dir, 'content')
  }

  _getNextPostNumber (year, month, day) {
    const pathToCheck = path.join(this.contentDir, year, month, day)
    fs.ensureDirSync(pathToCheck)

    const lastNum = fs.readdirSync(pathToCheck)
      .filter(f => fs.statSync(path.join(pathToCheck, f)).isDirectory())
      .sort().pop() || '00'

    return (parseInt(lastNum) + 1).toString().padStart(2, '0')
  }

  newPost ({ properties, commands }) {
    return this.limit(() => {
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

      if (meta.title === '') {
        meta.title = content.length > 15
          ? content.substring(0, 15) + '...'
          : content
      }

      // TODO: correctly parse location
      // and get more info

      if (properties.category) meta.tags = properties.category

      let slug = commands['mp-slug']
        ? commands['mp-slug'][0]
        : meta.title
          ? slugify(meta.title)
          : ''

      if (properties['bookmark-of']) {
        meta.categories = ['bookmarks']
        slug = ''
      } else {
        meta.categories = ['notes']
      }

      meta.properties = properties

      const num = this._getNextPostNumber(year, month, day)
      const alias = `/${year}/${month}/${day}/${num}/`
      const url = `${alias}${slug}`

      if (slug !== '') {
        meta.aliases = [alias]
      }

      const dirPath = path.join(this.contentDir, url)
      const indexPath = path.join(dirPath, 'index.md')
      const index = `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content}`

      fs.ensureDirSync(dirPath, { recursive: true })
      fs.writeFileSync(indexPath, index)

      this.gitCommit(`add ${url}`)
      this.gitPush()

      return `https://hacdias.com${url}`
    })
  }

  handleWebMention (webmention) {
    return this.limit(() => {
      const dataPath = path.join(
        this.contentDir,
        webmention.target.replace('https://hacdias.com/', '', 1),
        'data'
      )

      fs.ensureDirSync(dataPath)
      fs.writeFileSync(
        path.join(dataPath, 'index.md'),
        '---\nheadless: true\n---'
      )

      const dataFile = path.join(dataPath, 'webmentions.json')

      if (!fs.existsSync(dataFile)) {
        fs.outputJSONSync(dataFile, [webmention], {
          spaces: 2
        })
      } else {
        const arr = fs.readJSONSync(dataFile)
        const inArray = arr.filter(a => a['wm-id'] === webmention.post['wm-id']).length !== 0

        if (!inArray) {
          arr.push(webmention.post)
          fs.outputJSONSync(dataFile, arr, {
            spaces: 2
          })
        }
      }

      this.gitCommit(`webmention from ${webmention.post.url}`)
      this.gitPush()
    })
  }

  gitCommit (message) {
    let res = spawnSync('git', ['add', '-A'], { cwd: this.dir })
    if (res.error) throw res.error
    res = spawnSync('git', ['commit', '-m', message], { cwd: this.dir })
    if (res.error) throw res.error
  }

  gitPush () {
    const { error } = spawnSync('git', ['push'], { cwd: this.dir })
    if (error) throw error
  }
}
