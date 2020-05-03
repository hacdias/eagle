const ar = require('../utils/ar')

module.exports = ({ domain, user }) => {
  const url = new URL(domain)
  const resource = `acct:${user}@${url.hostname}`

  const object = {
    subject: resource,
    links: [
      {
        rel: 'self',
        type: 'application/activity+json',
        href: new URL(domain).origin + '/'
      }
    ]
  }

  return ar(async (req, res) => {
    if (req.query.resource !== resource) {
      return res.sendStatus(404)
    }

    return res.json(object)
  })
}
