const csv = require('@fast-csv/format')

module.exports = async function outputCheckins (out, hugo) {
  const stream = csv.format({
    headers: [
      'Date',
      'Name',
      'Country',
      'Region',
      'Locality',
      'Address',
      'Latitude',
      'Longitude',
      'Tags'
    ]
  })
  stream.pipe(out)

  for await (const { meta } of hugo.getAll({ keepOriginal: true })) {
    if (!meta.properties || !meta.properties.checkin) {
      continue
    }

    stream.write([
      meta.date.getTime(),
      meta.properties.checkin.properties.name,
      meta.properties.checkin.properties['country-name'],
      meta.properties.checkin.properties.region,
      meta.properties.checkin.properties.locality,
      meta.properties.checkin.properties['street-address'],
      meta.properties.checkin.properties.latitude,
      meta.properties.checkin.properties.longitude,
      meta.tags
    ])
  }

  stream.end()
}
