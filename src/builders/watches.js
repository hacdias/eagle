const fs = require('fs-extra')
const { join } = require('path')

module.exports = async ({ src, dst }) => {
  const historyFile = join(src, 'history.json')
  const history = await fs.readJSON(historyFile)

  const data = {}

  data.movies = Object.values(history.filter(({ type }) => type === 'movie')
    .reduce((acc, curr) => {
      if (acc[curr.movie.ids.slug]) {
        const watch = new Date(curr.watched_at)
        if (watch > acc[curr.movie.ids.slug].watched) {
          acc[curr.movie.ids.slug].watched = watch
        }
      } else {
        acc[curr.movie.ids.slug] = {
          title: curr.movie.title,
          watched: new Date(curr.watched_at),
          url: `https://trakt.tv/movies/${curr.movie.ids.slug}`
        }
      }

      return acc
    }, {}))
    .sort((a, b) => b.watched - a.watched)

  data.series = Object.values(history.filter(({ type }) => type === 'episode')
    .reduce((acc, curr) => {
      if (acc[curr.show.ids.slug]) {
        const watch = new Date(curr.watched_at)
        if (watch > acc[curr.show.ids.slug].watched) {
          acc[curr.show.ids.slug].watched = watch
        }
      } else {
        acc[curr.show.ids.slug] = {
          title: curr.show.title,
          watched: new Date(curr.watched_at),
          url: `https://trakt.tv/shows/${curr.show.ids.slug}`
        }
      }

      return acc
    }, {}))
    .sort((a, b) => b.watched - a.watched)

  await fs.outputJSON(dst, data, { spaces: 2 })
}
