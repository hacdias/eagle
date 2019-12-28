const createLikeOf = (properties) => {
  if (properties['like-of'].length !== 1) {
    throw new Error('invalid like of, length !== 1')
  }

  const url = properties['like-of'][0]
  const meta = {
    title: `Liked ${url}`,
    categories: ['likes']
  }

  // TODO: fetch 'like-of' source content

  return {
    meta,
    slug: false
  }
}

const createBookmark = (properties) => {
  const meta = {
    categories: ['bookmarks']
  }

  return {
    meta,
    slug: false
  }
}

module.exports = {
  createBookmark,
  createLikeOf
}
