
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
    request.type = `t-${body.h}`

    delete body.h
    delete body.access_token

    if (typeof body.action !== 'undefined') {
      throw new Error('cannot specify an action when creating a post')
    }

    for (const [key, value] in Object.entries(body)) {
      if (Array.isArray(value) && value.length === 0) {
        throw new Error('values in form-encoded input can only be numeric indexed arrays')
      }

      console.log(key)

      // TODO

      /*
       foreach($POST as $k=>$v) {

        if(is_array($v) && !isset($v[0]))
          return new Error('invalid_input', $k, 'Values in form-encoded input can only be numeric indexed arrays');
        if(is_array($v) && isset($v[0]) && is_array($v[0])) {
          return new Error('invalid_input', $k, 'Nested objects are not allowed in form-encoded requests');
        }
        // All values in mf2 json are arrays
        if(!is_array($v))
          $v = [$v];
        if(substr($k, 0, 3) == 'mp-') {
          $request->_commands[$k] = $v;
        } else {
          $request->_properties[$k] = $v;
        }
      }
      */
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
    const request = {}
    if (!Array.isArray(body.type)) {
      throw new Error('property "type" must be an array of microformat vocabularies')
    }

    request.action = 'create'
    request.type = body.type

    if (typeof body.properties !== 'object') {
      throw new Error('in JSON format, all properties must be specified in a properties object')
    }

    for (const [key, value] in Object.entries(body.properties)) {
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
      for (const key in Object.keys(request.update)) {
        if (typeof body[key] !== 'undefined') {
          // TODO: more validation
          request[key] = body[key]
        }
      }
    }

    return request
  }

  throw new Error('no micropub data was found in the request')
}

module.exports = {
  parseFormEncoded,
  parseJson
}
