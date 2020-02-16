#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const hugo = require('../src/services/hugo')(config.hugo)
const fs = require('fs-extra')
const { join } = require('path')
const crypto = require('crypto')

function sha256 (data) {
  return crypto.createHash('sha256').update(data).digest('hex')
}

async function migratePosts () {
  for await (const { post, meta, content } of hugo.getAll({ keepOriginal: true })) {
    if (!post.startsWith('/20')) continue

    let slug = ''
    let type

    if (meta.aliases) {
      delete meta.aliases
      slug = post.split('/').pop()
    }

    if (meta.categories) {
      type = meta.categories[0]
      delete meta.categories
    }

    const { post: newPost } = await hugo.newEntry({ meta, content, slug, type }, { keepOriginal: true })

    const oldPostPath = join(hugo.contentDir, post)
    const newPostPath = join(hugo.contentDir, newPost)

    const files = (await fs.readdir(oldPostPath))
      .filter(n => n !== 'index.md')

    for (const file of files) {
      await fs.move(join(oldPostPath, file), join(newPostPath, file))
    }

    await fs.remove(join(hugo.contentDir, post))

    console.log(`${post}/ ${newPost}`)
  }
}

;(async () => {
  const updates = [
    ['/blog/', '/articles/']
  ]

  for (const [oldPath, newPath] of updates) {
    let oldWm = sha256(oldPath) + '.json'
    let newWm = sha256(newPath) + '.json'

    oldWm = join(hugo.dataDir, 'mentions', oldWm)
    newWm = join(hugo.dataDir, 'mentions', newWm)

    if (await fs.exists(oldWm)) {
      await fs.move(oldWm, newWm)
    }
  }
})()
