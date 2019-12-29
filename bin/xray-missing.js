require('dotenv').config()

const { join } = require('path')
const fs = require('fs-extra')
const eagle = require('../src/eagle').fromEnvironment()
const yaml = require('js-yaml')

const getAllFiles = function (dirPath, arrayOfFiles) {
  const files = fs.readdirSync(dirPath)

  arrayOfFiles = arrayOfFiles || []

  files.forEach(function (file) {
    if (fs.statSync(dirPath + '/' + file).isDirectory()) {
      arrayOfFiles = getAllFiles(dirPath + '/' + file, arrayOfFiles)
    } else {
      arrayOfFiles.push(join(dirPath, '/', file))
    }
  })

  return arrayOfFiles
}

;(async () => {
  const path = join(eagle.hugoOpts.dir, 'content')

  const files = getAllFiles(path)
    .filter(p => p.endsWith('index.md'))

  for (const file of files) {
    const [frontmatter] = fs.readFileSync(file).toString().split('\n---')
    const meta = yaml.safeLoad(frontmatter)
    if (!meta.properties) {
      continue
    }

    let url = null

    if (meta.properties['like-of']) url = meta.properties['like-of'][0]
    if (meta.properties['repost-of']) url = meta.properties['repost-of'][0]
    if (meta.properties['in-reply-to']) url = meta.properties['in-reply-to'][0]

    if (!url) continue

    eagle._xrayAndSave(url)
  }
})()
