const express = require('express')
const app = express()
const port = 3000
const HugoManager = require('./hugo')

require('dotenv').config()

const micropub = require('./micropub')
const webmention = require('./webmention')

const hugo = new HugoManager({
  dir: require('path').join(__dirname, '../../hacdias.com')
})

app.use('/micropub', micropub({
  tokenReference: {
    me: 'https://hacdias.com/',
    endpoint: 'https://tokens.indieauth.com/token'
  },
  queryHandler: async (query) => {
    if (query.q === 'config') {
      return {
        'media-endpoint': 'https://api.hacdias.com/micropub'
      }
    }
    console.log(query)
    return {}
  },
  mediaHandler: async (files) => {
    console.log(files)
    return 'https://media.hacdias.com/file.jpg'
  },
  postHandler: async (data) => {
    console.log(JSON.stringify(data, null, 2))

    if (data.action === 'create') {
      return hugo.newPost(data)
    }

    return '/location/'
  }
}))

app.use('/webmention', webmention())

app.listen(port, () => console.log(`Listening on port ${port}!`))
