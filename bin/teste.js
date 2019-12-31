require('dotenv').config()

const eagle = require('../src/eagle').fromEnvironment()

;(async () => {
  const res = await eagle.twitter.tweet({ status: 'teste' })
  const url = `https://twitter.com/hacdias/status/${res.id_str}`

  console.log(url)
})()
