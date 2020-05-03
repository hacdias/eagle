module.exports = ({ services, domain }) => {
  const { hugo, git } = services

  return async (req, res, data) => {
    const post = data.url.replace(domain, '', 1)
    const { meta, content } = await hugo.getEntry(post)

    meta.expiryDate = new Date()
    await hugo.saveEntry(post, { meta, content })
    await git.commit(`delete ${post}`)

    res.sendStatus(200)
  }
}
