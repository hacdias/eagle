# Eagle

- [ ] Monitor external links for 404s, and replace with Web Archive'd when possible. Perhaps via slow running cron job with queue
- [ ] Command to check for consistency with AT Protocol
- [ ] SQLite instead of Bolt

## IndieAuth

- [ ] Implement OAuth 2.0 refresh tokens
  - [ ] https://indieauth.spec.indieweb.org/#refresh-tokens
  - [ ] Do not forget to add to `grant_types_supported`
- [ ] Keep all sessions on database, remove expired on cron job, allow for revoking session

## ATProto

- [ ] `site.standard` integration (https://standard.site/)
  - [ ] `site.standard.publication`
    - [ ] Upsert asynchronously
    - [ ] Support for custom theme
    - [ ] Support for custom icon
- [ ] Integrations Eagle --> AT Protocol
  - [ ] Recipes (/tags/recipe) with https://recipe.exchange/lexicons
  - [ ] Readings with Popfeed.social or Bookhive.buzz
    - NOTE: Popfeed allows tracking a lot more of Readings that maybe would give better UX for me. Potentially make this an inverse integration.
- [ ] Integration AT Protocol --> Eagle via jetstream listening:
  - [ ] Watches: movies, tv shows, (live performances)
  - [ ] Bookmarks
  - [ ] Readings: see note above
