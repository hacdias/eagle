const crypto = require('crypto')
const got = require('got')

const sendSigned = async (privateKey, obj, target) => {
  const url = new URL(target)
  const signer = crypto.createSign('sha256')
  const date = new Date()

  const body = JSON.stringify(obj)
  const digest = 'SHA-256=' + crypto.createHash('sha256').update(body).digest('base64')

  const stringToSign = `(request-target): post ${url.pathname}\nhost: ${url.host}\ndate: ${date.toUTCString()}\ndigest: ${digest}`
  signer.update(stringToSign)
  signer.end()
  const signature = signer.sign(privateKey).toString('base64')

  const header = `keyId="https://hacdias.com/#key",algorithm="rsa-sha256",headers="(request-target) host date digest",signature="${signature}"`

  await got.post(url.href, {
    body,
    headers: {
      'Content-Type': 'application/activity+json',
      Host: url.host,
      Date: date.toUTCString(),
      Digest: digest,
      Signature: header
    }
  })
}

module.exports = {
  sendSigned
}
