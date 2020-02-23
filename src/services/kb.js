const fs = require('fs-extra')
const { join } = require('path')
const yaml = require('js-yaml')
const slugify = require('slugify')

module.exports = function createKB ({ hugoDir, notesRepoDir, queue }) {
  const kbDir = join(hugoDir, 'content', 'kb')
  const notesDir = join(notesRepoDir, 'notes')

  return async () => {
    await fs.remove(kbDir)
    await fs.ensureDir(kbDir)

    await fs.outputFile(
      join(kbDir, '_index.md'),
    `---
title: Knowledge Base
emoji: 🧠
---`
    )

    const files = await fs.readdir(notesDir)

    for (const index of files) {
      const path = join(notesDir, index)
      if (fs.statSync(path).isDirectory()) {
        continue
      }

      const file = (await fs.readFile(path)).toString()
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
  }
}
