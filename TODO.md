# Eagle

- [ ] Monitor external links for 404s, and replace with Web Archive when possible
- [ ] Command to check if site.standard.document is consistent with the current website.

## Panel

- [ ] browse link to posts and sort from most recebnt to oldest, maybe option?
- [ ] Syndicate: add extra field for "status" iwht character count: that way I can customize what goes in.

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
