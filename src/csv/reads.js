const csv = require('@fast-csv/format')

module.exports = async function outputReads (out, hugo) {
  const stream = csv.format({
    headers: [
      'Status',
      'Date',
      'Author',
      'Name',
      'Rating',
      'UID',
      'Tags'
    ]
  })

  stream.pipe(out)

  for await (const { meta } of hugo.getAll({ keepOriginal: true })) {
    if (!meta.properties) {
      continue
    }

    if (!meta.categories || !meta.categories.includes('reads')) {
      continue
    }

    stream.write([
      meta.properties['read-status'],
      meta.date.getTime(),
      meta.properties['read-of'].properties.author,
      meta.properties['read-of'].properties.name,
      meta.properties['read-of'].properties.rating || 0,
      meta.properties['read-of'].properties.uid,
      meta.properties['bookmark-of'],
      meta.tags
    ])
  }

  stream.end()
}
