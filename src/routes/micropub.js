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

    // TODO: this must be source, call hugo.getSource()
    return {}
  },
  mediaHandler: async (files) => {
    console.log(files)
    return 'https://media.hacdias.com/file.jpg'
  },
  postHandler: async (data, origin) => {
    if (data.action === 'create') {
      return eagle.receiveMicropub(data, origin)
    }

    console.log(JSON.stringify(data, null, 2))
    return '/location/'
  }
})
