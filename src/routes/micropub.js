const micropub = require('../micropub')

const config = Object.freeze({
  'media-endpoint': 'https://api.hacdias.com/micropub',
  'syndicate-to': [
    {
      uid: 'https://twitter.com/',
      name: 'twitter.com'
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
  postHandler: async (data) => {
    if (data.action === 'create') {
      const url = await eagle.receiveMicropub(data)
      eagle.sendWebMentions(url)
      return url
    }

    console.log(JSON.stringify(data, null, 2))
    return '/location/'
  }
})
