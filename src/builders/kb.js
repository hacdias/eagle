const fs = require('fs-extra')
const { join } = require('path')
const yaml = require('js-yaml')
const slugify = require('slugify')

module.exports = async function buildKB ({ src, dst }) {
  // Delete all files except _index.md
  if (await fs.exists(dst)) {
    const files = await fs.readdir(dst)

    for (const file of files) {
      if (file === '_index.md') {
        continue
      }

      await fs.remove(join(dst, file))
    }
  }

  const files = await fs.readdir(src)

  for (const index of files) {
    const path = join(src, index)
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

    if (content.match(/(\$\$.*?\$\$|\$.*?\$)/g)) {
      meta.math = true
    }

    if (content.includes('```mermaid')) {
      meta.mermaid = true
    }

    // Replace wiki links by true links that work with Hugo
    content = content.replace(/\[\[(.*?)\]\]/g, (match, val) => {
      if (val.includes('|')) {
        const parts = val.split('|', 2)
        return `[${parts[0]}](/kb/${slugify(parts[1].toLowerCase())})`
      }

      return `[${val}](/kb/${slugify(val.toLowerCase())})`
    })

    await fs.outputFile(
      join(dst, `${slugify(meta.title.toLowerCase())}.md`),
      `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content.trim()}`
    )
  }
}
