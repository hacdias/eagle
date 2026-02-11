# Eagle

- [ ] Monitor external links for 404s, and replace with Web Archive'd when possible. Perhaps via slow running cron job with queue
- [ ] Command to check for consistency with AT Protocol
- [ ] SQLite instead of Bolt

## Panel

- [ ] Add way of adding Alt/Title to photo while creating post.
- [ ] Syndicate: add extra field for "status" with character count: that way I can customize what goes in.

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
  - [ ] `site.standard.document`
    - [ ] Double check if all conforms
  - [ ] Validate records against schema on publishing
- [ ] `app.bsky.feed.post` integration
  - [ ] Support for custom status instead of title + link for long form
  - [ ] Validate records against schema on publishing
- [ ] Integrations Eagle --> AT Protocol
  - [ ] Recipes (/tags/recipe) with https://recipe.exchange/lexicons
  - [ ] Readings with Popfeed.social or Bookhive.buzz
    - NOTE: Popfeed allows tracking a lot more of Readings that maybe would give better UX for me. Potentially make this an inverse integration.
- [ ] Integration AT Protocol --> Eagle via jetstream listening:
  - [ ] Watches: movies, tv shows, (live performances)
  - [ ] Bookmarks
  - [ ] Readings: see note above
