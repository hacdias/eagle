const got = require('got')
const stream = require('stream')
const { promisify } = require('util')
const pipeline = promisify(stream.pipeline)

module.exports = function createCdn ({ zone, key, base }) {
  const upload = async (stream, filename) => {
    if (!filename.startsWith('/')) {
      filename = '/' + filename
    }

    await pipeline(
      stream,
      got.stream.put(`https://storage.bunnycdn.com/${zone}${filename}`, {
        headers: {
          AccessKey: key
        }
      })
    )

    return base + filename
  }

  return Object.freeze({
    upload
  })
}
