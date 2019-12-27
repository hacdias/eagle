module.exports = (_, res) => {
  res.header('Content-Type', 'text/plain')
  res.send(`UserAgent: *
Disallow: /`)
}
