const fs = require('fs-extra')
const { join } = require('path')
const yaml = require('js-yaml')
const slugify = require('slugify')

const makeSlug = (wtv) => {
  return slugify(wtv, {
    lower: true,
    strict: true
  })
}

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
  const kb = {}

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

    const slug = makeSlug(meta.title)

    // Replace wiki links by true links that work with Hugo
    content = content.replace(/\[\[(.*?)\]\]/g, (match, val) => {
      let title = val
      let link = val

      if (val.includes('|')) {
        const parts = val.split('|', 2)
        title = parts[0]
        link = parts[1]
      }

      link = makeSlug(link)
      kb[link] = kb[link] || {}
      kb[link].refs = kb[link].refs || []
      kb[link].refs.push(slug)

      return `[${title}](/kb/${link}/)`
    })

    kb[slug] = kb[slug] || {}
    kb[slug].meta = meta
    kb[slug].content = content.trim()
  }

  for (const key in kb) {
    let { meta, content, refs } = kb[key]

    if (!meta) {
      continue
    }

    if (refs) {
      content += '\n## Referenced In\n\n'
      content += refs.map(url => `- [${kb[url].meta.title}](/kb/${url})`).join('\n').trim()
    }

    await fs.outputFile(
      join(dst, `${key}.md`),
      `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content.trim()}`
    )
  }
}
