require('dotenv').config()

const got = require('got')
const eagle = require('../src/eagle').fromEnvironment()

;(async () => {
  const res = await eagle.posse.twitter.like('1212337716999970818')
  console.log(res)
})()
