const got = require('got')
const debug = require('debug')('hugo:location')

const parseLocation = async (location) => {
  debug('got %o', location)

  if (!location.startsWith('geo:')) {
    debug('invalid, not parsing %o', location)
    return location
  }

  const loc = location.replace('geo:', '', 1)
    .split(';')[0]
    .split(',')

  try {
    const { body } = await got(`https://atlas.p3k.io/api/geocode?latitude=${loc[0]}&longitude=${loc[1]}`, {
      responseType: 'json'
    })

    const res = {
      type: 'h-adr',
      properties: {
        locality: body.locality,
        region: body.region,
        country: body.country,
        latitude: body.latitude,
        longitude: body.longitude
      }
    }

    debug('got location info %o', res)
    return res
  } catch (e) {
    debug('could not get info for %o: %s', location, e.toString())
    return location
  }
}

module.exports = parseLocation
