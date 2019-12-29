require('dotenv').config()

const Eagle = require('../src/eagle')

const eagle = Eagle.fromEnvironment()

eagle.sendWebMentions('https://hacdias.com/2019/12/28/06/url-structure/')
