const got = require('got')

module.exports = function createCdn ({ zone, key, base }) {
  const upload = async (data, filename) => {
    if (!filename.startsWith('/')) {
      filename = '/' + filename
    }

    await got.put(`https://storage.bunnycdn.com/${zone}${filename}`, {
      headers: {
        AccessKey: key
      },
      body: data
    })

    return base + filename
  }

  return Object.freeze({
    upload
  })
}
