const express = require('express')

module.exports = () => {
  const router = express.Router({
    caseSensitive: true,
    mergeParams: true
  })

  router.use(express.json())
  router.use(express.urlencoded({ extended: true }))

  router.post('/', (req, res) => {
    console.log(req.body)
  })

  return router
}
