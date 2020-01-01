const pluralize = require('pluralize')
const slugify = require('@sindresorhus/slugify')
const invert = require('lodash.invert')

const propertyToType = Object.freeze({
  rsvp: 'rsvp',
  'repost-of': 'repost',
  'like-of': 'like',
  'in-reply-to': 'reply',
  'bookmark-of': 'bookmark',
  'follow-of': 'follow',
  checkin: 'checkin',
  video: 'video',
  audio: 'audio',
  photo: 'photo'
})

const typeToProperty = invert(propertyToType)

const supportedTypes = Object.freeze([
  'repost', 'like', 'reply', 'bookmark', 'video', 'photo', 'note'
])

const hasURL = Object.freeze([
  'repost', 'like', 'reply', 'bookmark'
])

const buildSlugFor = Object.freeze([
  'note',
  'article'
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

class Micropub {
  // creates a new post.
  static createPost ({ properties, commands }) {
    const date = new Date()
    const type = postType(properties)

    if (!supportedTypes.includes(type)) {
      throw new Error(`type '${type} is not supported yet`)
    }

    const content = properties.content
      ? properties.content.join('\n').trim()
      : ''

    delete properties.content

    const meta = {
      title: properties.name
        ? properties.name.join(' ').trim()
        : '',
      categories: [pluralize(type)],
      date
    }

    delete properties.name

    const relatedURL = hasURL.includes(type)
      ? properties[typeToProperty[type]][0]
      : null

    if (meta.title === '') {
      meta.title = content.length > 15
        ? content.substring(0, 15).trim() + '...'
        : content
    }

    if (properties.category) {
      meta.tags = properties.category
    }

    meta.properties = properties

    const slug = Array.isArray(commands['mp-slug']) && commands['mp-slug'].length === 1
      ? commands['mp-slug'][0]
      : buildSlugFor.includes(type)
        ? meta.title
          ? slugify(meta.title)
          : ''
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
  static updatePost ({ meta, content }, { update }) {
    meta.properties = meta.properties || {}
    meta.tags = meta.tags || []

    for (const key in update.replace) {
      if (key === 'name') {
        meta.title = update.replace.name.join(' ').trim()
      } else if (key === 'category') {
        meta.tags = update.replace.category
      } else if (key === 'content') {
        content = update.content.join(' ').trim()
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
        meta.properties[key].push(...update.replace[key])
      }
    }

    if (Array.isArray(update.delete)) {
      for (const key of update.delete) {
        if (key === 'name' || key === 'content') {
          throw new Error(`cannot remove the ${key}`)
        } else if (key === 'category') {
          meta.tags = []
        } else if (key === 'content') {
        } else {
          delete meta.properties[key]
        }
      }
    } else {
      for (const [key, value] of Object.entries(update.delete)) {
        if (key === 'name' || key === 'content') {
          throw new Error(`cannot remove the ${key}`)
        } else if (key === 'category') {
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
}

module.exports = Micropub
