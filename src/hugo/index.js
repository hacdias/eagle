const fs = require('fs-extra')
const path = require('path')
const slugify = require('@sindresorhus/slugify')
const pLimit = require('p-limit')

const git = require('./git')
const { makePost } = require('./fs')
const { createBookmark, createLikeOf } = require('./creators')

module.exports = class HugoManager {
  constructor ({ dir }) {
    this.limit = pLimit(1)
    this.dir = dir
    this.contentDir = path.join(dir, 'content')
  }

  _newPost ({ properties, commands }) {
    const date = new Date()

    const content = properties.content
      ? properties.content.join('\n').trim()
      : ''

    delete properties.content

    let meta = {
      title: properties.name
        ? properties.name.join(' ').trim()
        : '',
      date
    }

    let res
    let hasSlug = true

    if (properties['bookmark-of']) {
      res = createBookmark(properties)
    }

    if (properties['like-of']) {
      res = createLikeOf(properties)
    }

    if (res) {
      meta = {
        ...meta,
        ...res.meta
      }
      hasSlug = res.slug
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

    if (properties.category) {
      meta.tags = properties.category
    }

    // TODO: correctly parse location
    // and get more info

    const slug = hasSlug
      ? commands['mp-slug']
        ? commands['mp-slug'][0]
        : meta.title
          ? slugify(meta.title)
          : ''
      : ''

    if (!meta.categories) {
      meta.categories = ['notes']
    }

    meta.properties = properties

    const url = makePost({
      date,
      meta,
      content,
      slug,
      contentDir: this.contentDir
    })

    git.commit(`add ${url}`, { cwd: this.dir })
    git.push({ cwd: this.dir })

    return `https://hacdias.com${url}`
  }

  newPost (data) {
    return this.limit(() => this._newPost(data))
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
        fs.outputJSONSync(dataFile, [webmention.post], {
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

      git.commit(`webmention from ${webmention.post.url}`, { cwd: this.dir })
      git.push({ cwd: this.dir })
    })
  }
}
