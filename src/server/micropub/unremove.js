module.exports = ({ services, domain }) => {
  const { hugo, git } = services

  return async (req, res, data) => {
    const post = data.url.replace(domain, '', 1)
    const entry = await hugo.getEntry(post)

    if (!entry.meta.expiryDate) {
      return res.sendStatus(400)
    }

    delete entry.meta.expiryDate

    await hugo.saveEntry(post, entry)
    await git.commit(`delete ${post}`)
  }
}
