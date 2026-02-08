# Eagle

- [ ] Monitor external links for 404s, and replace with Web Archive when possible
- [ ] Maybe rely on Goldmark ast parsing for webmentions detection? tests
  - [ ] For deleted posts see if can run `git` command for latest version of certain file

## IndieAuth

- [ ] Implement OAuth 2.0 refresh tokens
  - [ ] https://indieauth.spec.indieweb.org/#refresh-tokens
  - [ ] Do not forget to add to `grant_types_supported`
- [ ] Way of revoking existing token (store in Bolt)

## Panel

- [ ] Way to trigger syndication
- [ ] Way to trigger other actions

## ATProto

- [ ] Set text documents for site.standard.document
- [ ] Set content based on leaflet.pub (Goldmark with custom extension) e.g. https://pdsls.dev/at://did:plc:ragtjsm2j2vknwkz3zp4oxrd/site.standard.document/3m4qqzatka22o
- [ ] Set site.standard.document
- [ ] Add publication icon: e.g. https://pdsls.dev/at://did:plc:ragtjsm2j2vknwkz3zp4oxrd/site.standard.publication/3ly4hnkatvc2p
- [ ] Proper authentication
- [ ] Popfeed lexicons for watches, readings: https://popfeed.social/
  - [ ] Start with readings?
- [ ] Always POSSE
