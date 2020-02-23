#!/usr/bin/env node
'use strict'

require('dotenv').config()

const config = require('../src/config')()
const fs = require('fs-extra')
const { join } = require('path')
const yaml = require('js-yaml')
const slugify = require('slugify')
const DIR = '/Users/henriquedias/Code/hacdias/headache/notes'

;(async () => {
  const files = fs.readdirSync(DIR)
  const kbDir = join(config.hugo.dir, 'content/kb')

  await fs.remove(kbDir)
  await fs.ensureDir(kbDir)

  await fs.outputFile(
    join(kbDir, '_index.md'),
    `---
title: Knowledge Base
emoji: 🧠
---`
  )

  for (const index of files) {
    if (fs.statSync(join(DIR, index)).isDirectory()) {
      continue
    }

    const file = (await fs.readFile(join(DIR, index))).toString()
    let [frontmatter, content] = file.split('\n---')
    const meta = yaml.safeLoad(frontmatter)

    if ((meta.tags && meta.tags.includes('private')) || meta.deleted) {
      continue
    }

    meta.date = meta.modified
    meta.publishDate = meta.created

    delete meta.modified
    delete meta.created
    delete meta.pinned

    content = content.trim()
    if (content[0] === '#') {
      content = content.substring(content.indexOf('\n')).trim()
    }

    await fs.outputFile(
      join(kbDir, `${slugify(meta.title.toLowerCase())}.md`),
      `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content.trim()}`
    )
  }
})()
