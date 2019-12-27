require('dotenv').config()

const express = require('express')
const app = express()
const port = 3000
const HugoManager = require('./hugo')

const micropub = require('./routes/micropub')
const webmention = require('./routes/webmention')
const robots = require('./routes/robots')
const r404 = require('./routes/404')

const hugo = new HugoManager({
  dir: process.env.HUGO_DIR
})

app.use('/micropub', micropub({ hugo }))
app.use('/webmention', webmention({ hugo }))
app.get('/robots.txt', robots)
app.get(r404)

app.listen(port, () => console.log(`Listening on port ${port}!`))
