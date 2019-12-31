const got = require('got')
const debug = require('debug')('indieauth')

const getAuth = async (token, endpoint) => {
  debug('getting token info from %s', endpoint)

  const { body } = await got(endpoint, {
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`
    },
    responseType: 'json'
  })

  return body
}

module.exports = ({ endpoint, me }) => {
  return (req, res, next) => {
    let token

    if (req.headers.authorization) {
      token = req.headers.authorization.trim().split(/\s+/)[1]
    } else if (!token && req.body && req.body.access_token) {
      token = req.body.access_token
    }

    if (!token) {
      debug('missing authentication token')
      return res.status(401).json({
        error: 'unauthorized',
        error_description: 'missing authentication token'
      })
    }

    getAuth(token, endpoint)
      .then(data => {
        if (!data.me || !data.scope || Array.isArray(data.me) || Array.isArray(data.scope)) {
          debug('invalid response from endpoint')
          return res.status(403).json({
            error: 'forbidden',
            error_description: 'invalid response from endpoint'
          })
        }

        if (data.me !== me) {
          debug('user is not allowed')
          return res.status(403).json({
            error: 'forbidden',
            error_description: 'user not allowed'
          })
        }

        req.hasScope = (requiredScopes) => {
          const scopes = data.scope.split(' ')

          for (const scope of requiredScopes) {
            if (!scopes.includes(scope)) {
              debug('user does not have required scopes: %o, has %o', requiredScopes, scopes)
              res.status(401).json({
                error: 'insufficient_scope',
                error_description: `requires scopes: ${requiredScopes.join(', ')}`
              })

              return false
            }
          }

          return true
        }

        next()
      })
      .catch(e => {
        debug('internal error on auth: %s', e.toString())
        res.status(500).json({
          error: 'internal server error'
        })
      })
  }
}
