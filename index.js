const express = require('express')
const app = express()
const port = 3000

const micropub = require('./micropub')

app.use('/micropub', micropub({
  postHandler: (data) => {
    return '/location/'
  }
}))

app.listen(port, () => console.log(`Listening on port ${port}!`))
