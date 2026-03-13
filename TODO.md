# Eagle

- [ ] SQLite instead of Bolt
- [ ] Monitor external links for 404s, and replace with Web Archive'd when possible. Perhaps via slow running cron job with queue
  - [ ] After saving post for the **first time**, links should be queued so that they are archived at the **moment** the post is created (PostSaveHook integration?)
  - [ ] Cronjob to periodically check for 404s
- [ ] Webmention integration without relying on Webmention.io
- [ ] Command to check for consistency with AT Protocol

## IndieAuth

- [ ] Implement OAuth 2.0 refresh tokens
  - [ ] https://indieauth.spec.indieweb.org/#refresh-tokens
  - [ ] Do not forget to add to `grant_types_supported`

## ATProto

- [ ] `site.standard` integration (https://standard.site/)
  - [ ] Custom icon
- [ ] Integrations Eagle --> AT Protocol
  - [ ] Recipes (/tags/recipe) with https://recipe.exchange/lexicons
  - [ ] Readings with Popfeed.social or Bookhive.buzz
    - NOTE: Popfeed allows tracking a lot more of Readings that maybe would give better UX for me. Potentially make this an inverse integration.
- [ ] Integration AT Protocol --> Eagle via jetstream listening:
  - [ ] Watches: movies, tv shows, (live performances)
  - [ ] Bookmarks
  - [ ] Readings: see note above
