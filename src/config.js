module.exports = () => ({
  port: process.env.PORT || 3000,
  xrayEndpoint: process.env.XRAY_ENTRYPOINT,
  telegraphToken: process.env.TELEGRAPH_TOKEN,
  domain: new URL(process.env.DOMAIN).origin,
  webmentionIoSecret: process.env.WEBMENTION_IO_WEBHOOK_SECRET,
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
    chatId: parseInt(process.env.TELEGRAM_CHAT_ID)
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
  notes: {
    repositoryDir: process.env.NOTES_REPO,
    hookSecret: process.env.NOTES_SECRET
  },
  website: {
    hookSecret: process.env.HOOK_SECRET
  },
  trakt: {
    repositoryDir: process.env.TRAKT_DATA,
    secret: process.env.TRAKT_SECRET
  },
  activityPub: {
    store: process.env.ACTIVITYPUB_STORE,
    user: process.env.ACTIVITYPUB_USER
  }
})
