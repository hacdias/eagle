const debug = require('debug')('micropub')

const parseFormEncoded = (body) => {
  const request = {
    action: null,
    type: null,
    properties: {},
    commands: {},
    update: {
      replace: [],
      add: [],
      delete: []
    }
  }

  if (typeof body.h !== 'undefined') {
    request.action = 'create'
    request.type = `h-${body.h}`

    delete body.h
    delete body.access_token

    if (typeof body.action !== 'undefined') {
      throw new Error('cannot specify an action when creating a post')
    }

    for (let [key, value] of Object.entries(body)) {
      if (Array.isArray(value) && value.length === 0) {
        throw new Error('values in form-encoded input can only be numeric indexed arrays')
      }

      if (!Array.isArray(value)) {
        value = [value]
      }

      if (key.startsWith('mp-')) {
        request.commands[key] = value
      } else {
        request.properties[key] = value
      }
    }

    return request
  }

  if (typeof body.action !== 'undefined') {
    if (body.action === 'update') {
      throw new Error('micropub update actions require using the JSON syntax')
    }

    if (typeof body.url !== 'string') {
      throw new Error('micropub actions require a URL property')
    }

    request.action = body.action
    request.url = body.url

    return request
  }

  throw new Error('no micropub data was found in the request')
}

const parseJson = function (body) {
  const request = {
    action: null,
    type: null,
    properties: {},
    commands: {},
    update: {
      replace: [],
      add: [],
      delete: []
    }
  }

  if (typeof body.type !== 'undefined') {
    if (!Array.isArray(body.type)) {
      throw new Error('property "type" must be an array of microformat vocabularies')
    }

    request.action = 'create'
    request.type = body.type

    if (typeof body.properties !== 'object') {
      throw new Error('in JSON format, all properties must be specified in a properties object')
    }

    for (const [key, value] of Object.entries(body.properties)) {
      if (!Array.isArray(value) || value.length === 0) {
        throw new Error('property values in JSOn format must be arrays')
      }

      if (key.startsWith('mp-')) {
        request.commands[key] = value
      } else {
        request.properties[key] = value
      }
    }

    return request
  }

  if (typeof body.action !== 'undefined') {
    if (typeof body.url !== 'string') {
      throw new Error('Micropub actions require a URL property')
    }

    request.action = body.action
    request.url = body.url

    if (body.action === 'update') {
      for (const type of Object.keys(request.update)) {
        if (typeof body[type] !== 'undefined') {
          if (Array.isArray(typeof body[type])) {
            throw new Error(`${type} must not be an array`)
          }

          for (const [key, value] of Object.entries(body[type])) {
            if (!Array.isArray(value) && type !== 'delete') {
              throw new Error(`${key}.${type} must be an array`)
            }
          }

          request.update[type] = body[type]
        }
      }
    }

    return request
  }

  throw new Error('no micropub data was found in the request')
}

const parseFiles = (files) => {
  const allResults = {}

  ;['video', 'photo', 'audio', 'file'].forEach(type => {
    const result = []

    ;([].concat(files[type] || [], files[type + '[]'] || [])).forEach(file => {
      if (file.truncated) {
        debug('file was truncated')
        return
      }

      result.push({
        filename: file.originalname,
        buffer: file.buffer
      })
    })

    if (result.length) {
      allResults[type] = result
    }
  })

  return allResults
}

module.exports = {
  parseFormEncoded,
  parseJson,
  parseFiles
}
