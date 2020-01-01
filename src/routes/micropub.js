const micropub = require('../micropub')

const config = Object.freeze({
  'media-endpoint': 'https://api.hacdias.com/micropub',
  'syndicate-to': [
    {
      uid: 'twitter',
      name: 'Twitter'
    }
  ]
})

module.exports = ({ eagle }) => micropub({
  tokenReference: {
    me: 'https://hacdias.com/',
    endpoint: 'https://tokens.indieauth.com/token'
  },
  queryHandler: async (query) => {
    if (query.q === 'config') {
      return config
    }

    if (query.q === 'syndicate-to') {
      return { 'syndicate-to': config['syndicate-to'] }
    }

    return eagle.sourceMicropub(query.url)
  },
  mediaHandler: async (files) => {
    console.log(files)
    return 'https://media.hacdias.com/file.jpg'
  },
  postHandler: async (data, origin) => {
    switch (data.action) {
      case 'create':
        return eagle.receiveMicropub(data, origin)
      case 'update':
        return eagle.updateMicropub(data)
      case 'delete':
        throw new Error('not implemennted')
      default:
        throw new Error('invalid request')
    }
  }
})
