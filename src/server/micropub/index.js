const debug = require('debug')('eagle:server:micropub')
const express = require('express')
const multer = require('multer')
const ar = require('../../utils/ar')

const { parseJson, parseFormEncoded } = require('@hacdias/micropub-parser')
const indieauth = require('@hacdias/indieauth-middleware')

const createRemove = require('./remove')
const createUnremove = require('./unremove')
const createSource = require('./source')
const createUpdate = require('./update')
const createReceive = require('./receive')

// https://www.w3.org/TR/micropub

const badRequest = (res, reason, code = 400) => {
  debug('invalid request, code: %d; reason: %s', code, reason)
  res.status(code).json({
    error: 'invalid_request',
    error_description: reason
  })
}

const config = Object.freeze({
  'syndicate-to': [
    {
      uid: 'twitter',
      name: 'Twitter'
    }
  ]
})

module.exports = ({ services, domain, tokenReference }) => {
  const { queue, hugo, notify } = services

  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())
  router.use(express.urlencoded({ extended: true }))
  router.use(indieauth(tokenReference))

  const storage = multer.memoryStorage()
  const upload = multer({ storage: storage })

  router.use(upload.single('file'))

  const source = createSource({ services, domain })
  const update = createUpdate({ services, domain })
  const remove = createRemove({ services, domain })
  const receive = createReceive({ services, domain })
  const unremove = createUnremove({ services, domain })

  router.get('/', ar(async (req, res) => {
    debug('GET received; query: %o', req.query)

    switch (req.query.q) {
      case 'source':
        if (typeof req.query.url !== 'string') {
          return badRequest(res, 'url must be set on source query')
        }

        return res.json(await queue.add(() => source(req.query.url)))
      case 'config':
        return res.json(config)
      case 'syndicate-to':
        return res.json({ 'syndicate-to': config['syndicate-to'] })
      default:
        return badRequest(res, 'invalid query')
    }
  }))

  router.post('/', ar(async (req, res) => {
    debug('POST received')
    let request

    if (req.file) {
      return res.sendStatus(501)
    }

    try {
      if (req.is('json')) {
        request = parseJson(req.body)
      } else {
        request = parseFormEncoded(req.body)
      }
    } catch (e) {
      return badRequest(res, e.stack)
    }

    debug('POST transformed')

    switch (request.action) {
      case 'create':
        await queue.add(() => receive(req, res, request))
        break
      case 'update':
        await queue.add(() => update(req, res, request))
        break
      case 'delete':
        await queue.add(() => remove(req, res, request))
        break
      case 'undelete':
        await queue.add(() => unremove(req, res, request))
        break
      default:
        return badRequest(res, 'invalid request')
    }

    try {
      if (request.action === 'delete') {
        await hugo.buildAndClean()
      } else {
        await hugo.build()
      }
    } catch (err) {
      debug('could not rebuild website %s', err.stack)
      notify.sendError(err)
    }
  }))

  return router
}
