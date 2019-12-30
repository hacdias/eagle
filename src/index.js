require('dotenv').config()

const express = require('express')
const app = express()
const port = process.env.PORT || 3000

const Eagle = require('./eagle')

const micropub = require('./routes/micropub')
const webmention = require('./routes/webmention')
const robots = require('./routes/robots')
const r404 = require('./routes/404')

const eagle = Eagle.fromEnvironment()

app.use('/micropub', micropub({ eagle }))
app.use('/webmention', webmention({ eagle, secret: process.env.WEBMENTION_IO_WEBHOOK_SECRET }))
app.get('/robots.txt', robots)
app.use(r404)

app.listen(port, () => console.log(`Listening on port ${port}!`))
