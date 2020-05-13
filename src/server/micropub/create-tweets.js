const remark = require('remark')
const strip = require('strip-markdown')

const TXT_LIMIT = 134
const DELIMITERS = ['.', '?', '!', '…', ')']

const concat = (prev, curr) => {
  const last = prev.pop()
  if (!last) {
    prev.push(curr)
    return prev
  }
  const longer = last + ' ' + curr
  if (longer.length <= TXT_LIMIT) {
    prev.push(longer)
  } else {
    prev.push(last, curr)
  }
  return prev
}

const splitMore = (prev, curr) => {
  let parts = []

  if (curr.length > TXT_LIMIT) {
    parts = curr.match(/[\s\S]{1,134}(?!\S)/g)
      .map(t => t.trim())
      .map(t => !DELIMITERS.includes(t.charAt(t.length - 1)) ? t + '…' : t)
  } else {
    parts = [curr]
  }

  return [...prev, ...parts]
}

async function createTweets (contents, url) {
  const text = String(await remark().use(strip)
    .process(contents))
    .trim()
    .replace(/\.\.\./g, '…')

  var tweets = text
    .split(/(?<=[.?!…])\s/g)
    .map(t => t.trim())
    .filter(t => !!t)
    .reduce(splitMore, [])
    .reduce(concat, [])
    .map((t, i) => `${t} /${i + 1}`)

  tweets.forEach(t => console.log(t.length))

  const lastTweet = tweets.pop()

  if (url.length + lastTweet.length + 2 <= 140) {
    tweets.push(lastTweet + ' ' + url)
  } else {
    tweets.push(lastTweet)
    tweets.push('Read more at ' + url)
  }

  return tweets
}

module.exports = createTweets
