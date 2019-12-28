const like = (properties) => {
  if (properties['like-of'].length !== 1) {
    throw new Error('invalid like of, length !== 1')
  }

  const url = properties['like-of'][0]
  const meta = {
    title: `Liked ${url}`,
    categories: ['likes']
  }

  return {
    meta,
    url,
    slug: false
  }
}

const repost = (properties) => {
  if (properties['repost-of'].length !== 1) {
    throw new Error('invalid like of, length !== 1')
  }

  const url = properties['repost-of'][0]
  const meta = {
    title: `Reposted ${url}`,
    categories: ['reposts']
  }

  return {
    meta,
    url,
    slug: false
  }
}

const bookmark = (properties) => {
  const meta = {
    categories: ['bookmarks']
  }

  return {
    meta,
    slug: false
  }
}

const reply = (properties) => {
  if (properties['in-reply-to'].length !== 1) {
    throw new Error('invalid like of, length !== 1')
  }

  const meta = {
    categories: ['replies']
  }

  return {
    meta,
    url: properties['in-reply-to'][0],
    slug: false
  }
}

module.exports = {
  bookmark,
  like,
  repost,
  reply
}
