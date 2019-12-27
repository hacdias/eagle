const micropub = require('../micropub')

const config = Object.freeze({
  'media-endpoint': 'https://api.hacdias.com/micropub',
  'syndicate-to': []
})

module.exports = ({ hugo }) => micropub({
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
  postHandler: async (data) => {
    if (data.action === 'create') {
      return hugo.newPost(data)
    }

    console.log(JSON.stringify(data, null, 2))
    return '/location/'
  }
})
