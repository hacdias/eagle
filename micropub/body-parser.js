const reservedProperties = Object.freeze([
  'access_token',
  'q',
  'url',
  'update',
  'add',
  'delete'
])

const formEncodedKey = /\[([^\]]*)\]$/

const cleanEmptyKeys = (result) => {
  for (const key in result) {
    if (typeof result[key] === 'object' && Object.getOwnPropertyNames(result[key])[0] === undefined) {
      delete result[key]
    }
  }
}

const parseFormEncoded = (body) => {
  const result = {
    type: body.h ? ['h-' + body.h] : undefined,
    properties: {},
    mp: {}
  }

  if (body.h) {
    result.type = ['h-' + body.h]
    delete body.h
  }

  for (let key in body) {
    const rawValue = body[key]

    if (reservedProperties.indexOf(key) !== -1) {
      result[key] = rawValue
    } else {
      let targetProperty
      let value = rawValue
      let subKey

      while ((subKey = formEncodedKey.exec(key))) {
        if (subKey[1]) {
          const tmp = {}
          tmp[subKey[1]] = value
          value = tmp
        } else {
          value = [].concat(value)
        }
        key = key.slice(0, subKey.index)
      }

      if (key.indexOf('mp-') === 0) {
        key = key.substr(3)
        targetProperty = result.mp
      } else {
        targetProperty = result.properties
      }

      targetProperty[key] = [].concat(value)
    }
  }

  cleanEmptyKeys(result)
  return result
}

const parseJson = function (body) {
  const result = {
    properties: {},
    mp: {}
  }

  for (let key in body) {
    const value = body[key]

    if (reservedProperties.indexOf(key) !== -1 || ['properties', 'type'].indexOf(key) !== -1) {
      result[key] = value
    } else if (key.indexOf('mp-') === 0) {
      key = key.substr(3)
      result.mp[key] = [].concat(value)
    }
  }

  for (const key in body.properties) {
    if (['url'].indexOf(key) !== -1) {
      result[key] = result[key] || [].concat(body.properties[key])[0]
      delete body.properties[key]
    }
  }

  cleanEmptyKeys(result)
  return result
}

module.exports = {
  parseFormEncoded,
  parseJson
}
