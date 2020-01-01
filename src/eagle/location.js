const got = require('got')
const debug = require('debug')('eagle:location')

const parseGeoAddress = async (location) => {
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
    debug('could not get info for %o: %s', location, e.stack)
    throw e
  }
}

module.exports = class LocationService {
  // TODO: save compass data location

  async updateEntry (meta) {
    if (meta.properties.location) {
      const loc = await Promise.all(
        meta.properties
          .location
          .map(loc => parseGeoAddress(loc))
      )

      meta.properties.location = loc
    } else {
      // TODO: also check my GPS logs
    }
  }
}
