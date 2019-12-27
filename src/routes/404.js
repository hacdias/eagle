module.exports = (_, res) => {
  res.header('Content-Type', 'text/plain')
  res.status(404).send("Darlings, there's nothing to see here! Muah 💋")
}
