# Eagle

> Is there something that you think that could be pulled over to an external module?
> Please [let me know](https://github.com/hacdias/eagle-go/issues/new)!

This powers my website. It is open-source. However, I won't be supporting other people's use
cases as this is just a personal project for personal use. If you're interested in doing
something similar, I encourage you to take a look at the code.

This repository replaces the old, JavaScript based, [API](https://github.com/hacdias/eagle-js).

## Update Plan

1. Remove Micropub service
2. Create endpoints for:
   1. Creation / edition
   2. Deletion
3. Protect endpoints with basic auth
4. Improve current search functionality and endpoint
   1. Allow more personalization on the website
   2. Allow the URL to indicate what we are searching
5. Stop relying on GoodReads for my reading section. Streamline reads file and make it easy to edit by myself (add custom link possibility for reviews).
6.  Improve bookmarks section using posts again. Format: /bookmarks/{slug}. Show them table like. Allow for search.
7.  CLI for local management.
8.  Solve newsletter/goodbye and thanks

### Notes

- Services must use local syncs. There must be some kind of global sync that allows to avoid calling hugo.Build while other operations are being some.
- Always commit specific files. Services that change files should take the storae service.)
- Detect file types via middleware.
- Consider using .html instead of / with .html stripped in the end URL
- Move all pictures to a different place and flatten content directory to simple markdown files.
- Find a different place to put my activitypub data.
- Store must be called by services that use it

## GOAL: support ActivityPub

- Make Inbox
- Make Outbox (?)

## License

MIT © Henrique Dias
