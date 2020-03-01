const got = require('got')

async function get (url) {
  const { body: actor } = await got(url, {
    headers: {
      Accept: 'application/activity+json',
      'Accept-Charset': 'utf-8'
    },
    responseType: 'json'
  })

  return {
    url: url,
    inbox: actor.inbox
  }
}

module.exports = {
  get
}
