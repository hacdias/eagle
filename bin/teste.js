require('dotenv').config()

const eagle = require('../src/eagle').fromEnvironment()

;(async () => {
  console.log(await eagle.hugo.getEntry('/2019/12/24/01/own-your-data/'))
})()
