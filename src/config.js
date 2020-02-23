module.exports = () => ({
  port: process.env.PORT || 3000,
  xrayEntrypoint: process.env.XRAY_ENTRYPOINT,
  telegraphToken: process.env.TELEGRAPH_TOKEN,
  domain: process.env.DOMAIN,
  hugo: {
    dir: process.env.HUGO_DIR,
    publicDir: process.env.HUGO_PUBLIC_DIR
  },
  twitter: {
    apiKey: process.env.TWITTER_API_KEY,
    apiSecret: process.env.TWITTER_API_SECRET,
    accessToken: process.env.TWITTER_ACCESS_TOKEN,
    accessTokenSecret: process.env.TWITTER_ACCESS_TOKEN_SECRET
  },
  telegram: {
    token: process.env.TELEGRAM_TOKEN,
    chatID: parseInt(process.env.TELEGRAM_CHAT_ID)
  },
  tokenReference: {
    me: process.env.TOKEN_REF_ME,
    endpoint: process.env.TOKEN_REF_ENDPOINT
  },
  bunny: {
    zone: process.env.BUNNY_ZONE,
    key: process.env.BUNNY_KEY,
    base: process.env.BUNNY_BASE
  },
  notesRepo: process.env.NOTES_REPO
})
