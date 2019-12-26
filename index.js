const express = require('express')
const app = express()
const port = 3000

require('dotenv').config()

const micropub = require('./micropub')

app.use('/micropub', micropub({
  postHandler: (data) => {
    console.log(JSON.stringify(data, null, 2))
    return '/location/'
  }
}))

app.listen(port, () => console.log(`Listening on port ${port}!`))
