const parseLocation = require('./location')
const slugify = require('@sindresorhus/slugify')

const types = Object.freeze([
  {
    prop: 'like-of',
    title: 'Liked',
    category: 'likes'
  },
  {
    prop: 'repost-of',
    title: 'Reposted',
    category: 'reposts'
  },
  {
    prop: 'in-reply-to',
    title: 'Replied to',
    category: 'replies'
  }
])

const parseType = (properties) => {
  // TODO: check if matches more than once, then abort.
  for (const { prop, title, category } of types) {
    if (!properties[prop]) {
      continue
    }

    if (properties[prop].length !== 1) {
      throw new Error(`invalid ${prop} length !== 1`)
    }

    const url = properties[prop][0]
    const meta = {
      title: `${title} ${url}`,
      categories: [category]
    }

    return {
      meta,
      relatedTo: {
        url,
        prop
      }
    }
  }
}

module.exports = async ({ properties, commands }) => {
  const date = new Date()

  const content = properties.content
    ? properties.content.join('\n').trim()
    : ''

  delete properties.content

  let meta = {
    title: properties.name
      ? properties.name.join(' ').trim()
      : '',
    date
  }

  const titleWasEmpty = meta.title === ''
  let relatedTo = null
  let buildSlug = true

  if (properties['bookmark-of']) {
    meta.categories = ['bookmarks']
    buildSlug = false
  } else {
    const res = parseType(properties)
    if (res) {
      meta = {
        ...meta,
        ...res.meta
      }
      relatedTo = res.relatedTo
      buildSlug = false
    } else {
      meta.categories = ['notes']
    }
  }

  if (meta.title === '') {
    meta.title = content.length > 15
      ? content.substring(0, 15).trim() + '...'
      : content
  }

  delete properties.name

  if (meta.title === '' && content === '') {
    throw new Error('must have title or content')
  }

  if (properties.category) {
    meta.tags = properties.category
  }

  if (properties.location) {
    properties.location = await Promise.all(
      properties
        .location
        .map(loc => parseLocation(loc))
    )
  } else {
    // TODO: also check my GPS logs
  }

  const slug = buildSlug
    ? commands['mp-slug']
      ? commands['mp-slug'][0]
      : meta.title
        ? slugify(meta.title)
        : ''
    : ''

  meta.properties = properties

  return {
    meta,
    content,
    slug,
    relatedTo,
    titleWasEmpty
  }
}
