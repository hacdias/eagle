const badRequest = (res, reason, code) => {
  res.status(code || 400).json({
    error: 'invalid_request',
    error_description: reason
  })
}

module.exports = {
  badRequest
}
