const transformer = require('./transformer')

module.exports = ({ services, domain }) => {
  const { hugo, git } = services

  return async (req, res, data) => {
    const post = data.url.replace(domain, '', 1)
    const entry = transformer.updatePost(await hugo.getEntry(post), data)

    await hugo.saveEntry(post, entry)
    await git.commit(`update ${post}`)

    res.redirect(200, data.url)
  }
}
