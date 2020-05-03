const crypto = require('crypto')

module.exports = (data) => {
  return crypto.createHash('sha256').update(data).digest('hex')
}
