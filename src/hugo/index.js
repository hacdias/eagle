const fs = require('fs-extra')
const path = require('path')
const slugify = require('@sindresorhus/slugify')
const pLimit = require('p-limit')
const crypto = require('crypto')
const debug = require('debug')('hugo')

const git = require('./git')
const create = require('./creators')
const { makePost } = require('./fs')
const parseLocation = require('./location')
const xray = require('./xray')

module.exports = class HugoManager {
  constructor ({ dir }) {
    this.limit = pLimit(1)
    this.dir = dir
    this.contentDir = path.join(dir, 'content')
  }

  async _xrayAndSave (url) {
    debug('gonna xray %s', url)

    try {
      const sha256 = crypto.createHash('sha256').update(url).digest('hex')
      const rxayDir = path.join(this.dir, 'data', 'xray')
      const xrayFile = path.join(rxayDir, `${sha256}.json`)

      if (!await fs.existsSync(xrayFile)) {
        const data = await xray(url)

        if (data.code !== 200) {
          return
        }

        await fs.outputJSON(xrayFile, data.data, {
          spaces: 2
        })

        debug('%s successfully xrayed', url)
      } else {
        debug('%s already xrayed', url)
      }
    } catch (e) {
      debug('could not xray %s: %s', url, e.toString())
    }
  }

  async _newPost ({ properties, commands }) {
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

    // TODO: check if matches more than once, then abort.
    if (properties['bookmark-of']) {
      res = create.bookmark(properties)
    } else if (properties['like-of']) {
      res = create.like(properties)
    } else if (properties['repost-of']) {
      res = create.repost(properties)
    } else if (properties['in-reply-to']) {
      res = create.reply(properties)
    } else {
      res = {
        meta: {
          categories: ['notes']
        },
        slug: true
      }
    }

    meta = {
      ...meta,
      ...res.meta
    }

    if (meta.title === '') {
      meta.title = content.length > 15
        ? content.substring(0, 15).trim() + '...'
        : content
    }

    if (res.url) {
      await this._xrayAndSave(res.url)
      // TODO: check for new title
    }

    hasSlug = res.slug

    delete properties.name

    if (meta.title === '' && content === '') {
      throw new Error('must have title or content')
    }

    if (properties.category) {
      meta.tags = properties.category
    }

    if (properties.location) {
      properties.location = await Promise.all(
        properties
          .location
          .map(loc => parseLocation(loc))
      )
    } else {
      // TODO: also check my GPS logs
    }

    const slug = hasSlug
      ? commands['mp-slug']
        ? commands['mp-slug'][0]
        : meta.title
          ? slugify(meta.title)
          : ''
      : ''

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
