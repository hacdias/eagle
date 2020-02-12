const csv = require('@fast-csv/format')

module.exports = async function outputBookmarks (out, hugo) {
  const stream = csv.format({
    headers: [
      'Title',
      'Date Added',
      'URL',
      'Tags'
    ]
  })
  stream.pipe(out)

  for await (const { meta } of hugo.getAll({ keepOriginal: true })) {
    if (!meta.properties) {
      continue
    }

    if (!meta.categories || !meta.categories.includes('bookmarks')) {
      continue
    }

    stream.write([meta.title, meta.date.getTime(), meta.properties['bookmark-of'], meta.tags])
  }

  stream.end()
}
