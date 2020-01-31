const { ar } = require('./utils')
const got = require('got')

async function jsonGot (url) {
  const { body } = await got(url, {
    responseType: 'json'
  })

  return body
}

const fToC = (f) => (f - 32) * 5 / 9

let cache = null

module.exports = () => ar(async (req, res) => {
  if (cache) {
    res.json(cache)
  }

  const compass = await jsonGot(`${process.env.COMPASS_ENDPOINT}/api/last?token=${process.env.COMPASS_TOKEN}`)

  const data = {
    time: Date.now(),
    battery: compass.data.properties.battery_level,
    plugged: compass.data.properties.battery_state === 'plugged',
    weather: {}
  }

  const [long, lat] = compass.data.geometry.coordinates
  const atlas = await jsonGot(`https://atlas.p3k.io/api/timezone?latitude=${lat}&longitude=${long}`)
  const weather = await jsonGot(`https://atlas.p3k.io/api/weather?latitude=${lat}&longitude=${long}&apikey=${process.env.DARK_SKY_TOKEN}`)

  data.time = atlas.localtime

  data.weather = {
    value: fToC(weather.temp.num),
    desc: weather.description
  }

  if (!cache) {
    res.json(data)
  }

  cache = data
})
