const fs = require('fs-extra')
const { join } = require('path')
const yaml = require('js-yaml')

module.exports = async function buildKB ({ src, dst }) {
  // Cleanup destination
  await fs.remove(dst)
  await fs.ensureDir(dst)

  const files = await fs.readdir(src)
  const kb = {}

  for (const index of files) {
    const path = join(src, index)
    if (fs.statSync(path).isDirectory() || !index.endsWith('.md')) {
      continue
    }

    const file = (await fs.readFile(path)).toString()
    const [frontmatter] = file.split('\n---', 2)
    let content = file.replace(frontmatter, '').trim()
    const meta = yaml.safeLoad(frontmatter)

    if (meta.private) {
      continue
    }

    if (content.match(/(\$\$.*?\$\$|\$.*?\$)/g)) {
      meta.math = true
    }

    if (content.includes('```mermaid')) {
      meta.mermaid = true
    }

    content = content.replace(/\[(.*?)\]\((.*?)\)/g, (match, title, link) => {
      if (!link.includes('.md')) {
        return match
      }

      const cleanLink = '/' + link.replace('.md', '/')
      const to = link.includes('#') ? link.split('#')[0] : link

      kb[to] = kb[to] || {}
      kb[to].refs = kb[to].refs || []
      kb[to].refs.push(index)

      return `[${title}](${cleanLink})`
    })

    kb[index] = kb[index] || {}
    kb[index].meta = meta
    kb[index].content = content.trim()
  }

  for (const file in kb) {
    let { meta, content, refs } = kb[file]

    if (!meta) {
      continue
    }

    if (refs) {
      content += '\n## Referenced In\n\n'
      content += refs.map(url => `- [${kb[url].meta.title}](/${url.replace('.md', '')}/)`).join('\n').trim()
    }

    await fs.outputFile(
      join(dst, file),
      `---\n${yaml.safeDump(meta, { sortKeys: true })}\n${content.trim()}`
    )
  }
}
