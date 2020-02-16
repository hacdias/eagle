const { ar } = require('./utils')

module.exports = () => ar(async (req, res) => {
  if (req.query.resource !== 'acct:hacdias@hacdias.com') {
    return res.sendStatus(404)
  }

  return res.json({
    subject: 'acct:hacdias@hacdias.com',
    links: [
      {
        rel: 'self',
        type: 'application/activity+json',
        href: 'https://hacdias.com/'
      }
    ]
  })
})
