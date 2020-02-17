const { ar } = require('./utils')

module.exports = () => ar(async (req, res) => {
  res.sendStatus(405)
})
