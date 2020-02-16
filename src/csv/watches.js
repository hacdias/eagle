const csv = require('@fast-csv/format')

module.exports = async function (out, hugo) {
  const stream = csv.format({
    headers: [
      'Date',
      'Title',
      'Movie'
    ]
  })

  stream.pipe(out)

  for await (const { meta } of hugo.getAll({ keepOriginal: true })) {
    if (!meta.properties || !meta.properties['watch-of']) {
      continue
    }

    stream.write([
      meta.date.getTime(),
      meta.properties['watch-of'].properties.title,
      !meta.properties['watch-of'].properties.show
    ])
  }

  stream.end()
}
