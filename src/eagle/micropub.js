const pluralize = require('pluralize')
const invert = require('lodash.invert')

const propertyToType = Object.freeze({
  rsvp: 'rsvp',
  'repost-of': 'repost',
  'like-of': 'like',
  'in-reply-to': 'reply',
  'bookmark-of': 'bookmark',
  'follow-of': 'follow',
  'read-of': 'read',
  checkin: 'checkin',
  video: 'video',
  audio: 'audio',
  photo: 'photo'
})

const typeToProperty = invert(propertyToType)

const supportedTypes = Object.freeze([
  'repost', 'like', 'reply', 'bookmark', 'video', 'photo', 'note', 'read'
])

const hasURL = Object.freeze([
  'repost', 'like', 'reply', 'bookmark'
])

// https://www.w3.org/TR/post-type-discovery/
// Code highly based on https://github.com/aaronpk/XRay/blob/5b2b4f31425ffe9c68833a26903fd1716b75717a/lib/XRay/PostType.php
const postType = (post) => {
  if (['event', 'recipe', 'review'].includes(post.type)) {
    return post.type
  }

  for (const prop in propertyToType) {
    if (typeof post[prop] !== 'undefined') {
      return propertyToType[prop]
    }
  }

  let content = ''
  if (typeof post.content !== 'undefined') {
    content = post.content.text
  } else if (typeof post.summary !== 'undefined') {
    content = post.summary
  }

  if (typeof post.name === 'undefined' || post.name.join(' ').trim() === '') {
    return 'note'
  }

  // Collapse all sequences of internal whitespace to a single space (0x20) character each
  const name = post.name.join(' ').trim().replace(/\s+/, ' ')
  content = content.replace(/\s+/, ' ')

  // If this processed "name" property value is NOT a prefix of the
  // processed "content" property, then it is an article post.
  if (content.indexOf(name) === -1) {
    return 'article'
  }

  return 'note'
}

// creates a new post.
const createPost = ({ properties, commands }) => {
  const date = properties.published
    ? new Date(properties.published)
    : new Date()

  delete properties.published
  const type = postType(properties)

  if (type === 'read') {
    // delete unwanted summary from indiebookclub.biz
    delete properties.summary
  }

  if (!supportedTypes.includes(type)) {
    throw new Error(`type '${type} is not supported yet`)
  }

  const content = properties.content
    ? properties.content.join('\n').trim()
    : ''

  delete properties.content

  const meta = {
    categories: [pluralize(type)],
    date
  }

  if (properties.name) {
    meta.title = properties.name.join(' ').trim()
  }

  delete properties.name

  const relatedURL = hasURL.includes(type)
    ? properties[typeToProperty[type]][0]
    : null

  if (properties.category) {
    meta.tags = properties.category
  }

  meta.properties = properties

  const slug = Array.isArray(commands['mp-slug']) && commands['mp-slug'].length === 1
    ? commands['mp-slug'][0]
    : ''

  return {
    meta,
    content,
    type,
    slug,
    relatedURL
  }
}

// Update updates a { meta, content } post with the
// update properties and returns a { meta, content }
// post.
const updatePost = ({ meta, content }, { update }) => {
  meta.properties = meta.properties || {}
  meta.tags = meta.tags || []
  update.replace = update.replace || {}
  update.add = update.add || {}
  update.delete = update.delete || {}

  for (const key in update.replace) {
    if (key === 'name') {
      meta.title = update.replace.name.join(' ').trim()
    } else if (key === 'category') {
      meta.tags = update.replace.category
    } else if (key === 'content') {
      content = update.replace.content.join(' ').trim()
    } else {
      meta.properties[key] = update.replace[key]
    }
  }

  for (const key in update.add) {
    if (key === 'name') {
      throw new Error('cannot add a new name')
    } else if (key === 'category') {
      meta.tags.push(...update.add.category)
    } else if (key === 'content') {
      content += update.add.join(' ').trim()
    } else {
      meta.properties[key] = meta.properties[key] || []
      meta.properties[key].push(...update.add[key])
    }
  }

  if (Array.isArray(update.delete)) {
    for (const key of update.delete) {
      if (key === 'category') {
        meta.tags = []
      } else if (key === 'content') {
        content = ''
      } else {
        delete meta.properties[key]
      }
    }
  } else {
    for (const [key, value] of Object.entries(update.delete)) {
      if (key === 'content') {
        content = ''
      } if (key === 'category') {
        meta.tags = meta.tags.filter(tag => !value.includes(tag))
      } else {
        meta.properties[key] = meta.properties[key]
          .filter(tag => !value.includes(tag))
      }
    }
  }

  meta.properties.category = meta.tags
  return { meta, content }
}

module.exports = {
  createPost,
  updatePost
}
