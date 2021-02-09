# Eagle

> Is there something that you think that could be pulled over to an external module?
> Please [let me know](https://github.com/hacdias/eagle-go/issues/new)!

This powers my website. It is open-source. However, I won't be supporting other people's use
cases as this is just a personal project for personal use. If you're interested in doing
something similar, I encourage you to take a look at the code.

This repository replaces the old, JavaScript based, [API](https://github.com/hacdias/eagle-js).

## Update Plan

1. X
2. X
4. Remove Micropub service
5. Create basic auth protected endpoints for:
   1. Creation (uses archetypes and allows to set URL as well as syndication)
   2. Edition
   3. Deletion
6. Streamline webmentions into a new format. Store in data (?) folder, path based - call them interactions. Cleanup format.
7. Improve current search functionality and endpoint
   1. Allow more personalization on the website
   2. Allow the URL to indicate what we are searching
8. Stop relying on GoodReads for my reading section. Streamline reads file and make it easy to edit by myself (add custom link possibility for reviews).
9. Improve bookmarks section using posts again. Format: /bookmarks/{slug}. Show them table like. Allow for search.
10. CLI for local management.
11. Solve newsletter/goodbye and thanks
12. X

### Notes

- Services must use local syncs. There must be some kind of global sync that allows to avoid calling hugo.Build while other operations are being some.
- Always commit specific files.
- Services must have all services in the same root. Like services.build (maybe move services to the root and call it Eagle!)
- Detect file types via middleware.
- Consider using .html instead of / with .html stripped in the end URL
- Move all pictures to a different place and flatten content directory to simple markdown files.
- private webmentions should be LOGGED and SENT by ntification service. Not stored on disk.
- Find a different place to put my activitypub data.
- Store must be called by services that use it

## GOAL: support ActivityPub

- Make Inbox
- Make Outbox (?)

## License

MIT © Henrique Dias
