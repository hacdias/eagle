require('dotenv').config()

const got = require('got')
const eagle = require('../src/eagle').fromEnvironment()

;(async () => {
  const res = await eagle.twitter.like('1211725899047034881')
  console.log(res)
})()
