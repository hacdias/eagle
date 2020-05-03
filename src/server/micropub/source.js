module.exports = ({ services, domain }) => {
  const { hugo } = services

  return async (url) => {
    if (!url.startsWith(domain)) {
      throw new Error('invalid request')
    }

    const post = url.replace(domain, '', 1)
    const { meta, content } = await hugo.getEntry(post)

    const entry = {
      type: ['h-entry'],
      properties: meta.properties
    }

    if (meta.title) {
      entry.properties.name = [meta.title]
    }

    if (meta.tags) {
      entry.properties.category = meta.tags
    }

    if (content) {
      entry.properties.content = [content]
    }

    if (meta.date) {
      entry.properties.published = [meta.date]
    }

    return entry
  }
}
