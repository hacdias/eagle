# Eagle

- [ ] Monitor external links for 404s, and replace with Web Archive when possible
- [ ] Maybe rely on Goldmark ast parsing for webmentions detection? tests
  - [ ] For deleted posts see if can run `git` command for latest version of certain file

## IndieAuth

- [ ] Implement OAuth 2.0 refresh tokens
  - [ ] https://indieauth.spec.indieweb.org/#refresh-tokens
  - [ ] Do not forget to add to `grant_types_supported`
- [ ] Way of revoking existing token (store in Bolt)

## ATProto

- [ ] Reuse authentication
- [ ] `site.standard.document`
  - [ ] Define `content` based on https://leaflet.pub's lexicons (can use Goldmark as parser with custom renderer)
- [ ] `site.standard.publication`
  - [ ] Upsert asynchronously
  - [ ] Add custom theme
  - [ ] Add custom icon
- [ ] Keep an eye on:
  - [ ] https://popfeed.social's lexicons for watches (movies, series), readings, perhaps music
  - [ ] https://bookhive.buzz/
  - [ ] https://teal.fm/, https://rocksky.app/
  - [ ] https://recipe.exchange/lexicons/
  - [ ] https://dropanchor.app's lexicons for Swarm-like check-ins
  - [ ] Listen on events (possible) and automatically update website?
- [ ] See way to enable auto POSSE'ing without manually triggering it every time
