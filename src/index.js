require('dotenv').config()

const express = require('express')
const app = express()
const port = process.env.PORT || 3000

const Eagle = require('./eagle')

const micropub = require('./routes/micropub')
const webmention = require('./routes/webmention')
const now = require('./routes/now')

const eagle = Eagle.fromEnvironment()

const tokenReference = {
  me: 'https://hacdias.com/',
  endpoint: 'https://tokens.indieauth.com/token'
}

app.use(express.json())

app.use('/micropub', micropub({
  eagle,
  tokenReference
}))

app.use('/webmention', webmention({
  eagle,
  secret: process.env.WEBMENTION_IO_WEBHOOK_SECRET
}))

app.get('/now', now())

app.get('/robots.txt', (_, res) => {
  res.header('Content-Type', 'text/plain')
  res.send('UserAgent: *\nDisallow: /')
})

app.use((_, res) => {
  res.header('Content-Type', 'text/plain')
  res.status(404).send("Darlings, there's nothing to see here! Muah 💋")
})

app.listen(port, () => console.log(`Listening on port ${port}!`))
