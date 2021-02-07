# Eagle

> Is there something that you think that could be pulled over to an external module?
> Please [let me know](https://github.com/hacdias/eagle-go/issues/new)!

This powers my website. It is open-source. However, I won't be supporting other people's use
cases as this is just a personal project for personal use. If you're interested in doing
something similar, I encourage you to take a look at the code.

This repository replaces the old, JavaScript based, [API](https://github.com/hacdias/eagle-js).

## Notes

## GOAL: support ActivityPub

- Make Inbox
- Make Outbox (?)

## GOAL: local syncs

- Use local syncs per service. Services should return the files changed!
- Then, we can commit specific files!

## License

MIT © Henrique Dias

# My Own CMS

**Inspirations**
- https://github.com/hacdias/eagle
- https://git.jlel.se/jlelse/GoBlog
- https://github.com/Xe/site

**Technology** either Go or Rust (https://www.arewewebyet.org/)

**Features** (or the lack of)
- Dynamic, thus IPFS support will be removed.
- Online admin backend:
	- **Remove** support for Micropub. Only I use it and I prefer to edit the files entirely.
	- Support for removing/monitoring comments.
- Webmentions
	- Which I should standardize to some comments backend
	- Comment API so people can directly comment on the website
- Separate comments data from the actual posts data / simplify.
- (Better) search!
- Bookmarking support.
- Readings support (remove GoodReads and delete account)
- Cache.
- CLI
	- similar to nb
	- Encrypted notes/posts that are not published (OR ONLY AVAILABLE AFTER LOGIN :)))) )
	- Offline first? Or online first (using the website API?)
	- `add` / `a`
	- `list` / `ls`
	- `edit` / `e` (opens on default editor and "knows" when editor was closed to commit the changes)
	- `search` / `s`
	- Each comment does git commit + push (add flag)
 