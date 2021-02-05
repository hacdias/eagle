---
publishDate: "2020-07-18T09:25:47.181Z"
tags:
- meta
- activitypub
---

I've been in spring cleanup mode since yesterday and I started on GitHub and ended up archiving and deprecating dozens of repos that were either unmaintained or actually supposed to be deprecated.

On this website, I just removed the ActivityPub support of my website because of two reasons: it had bugs and wasn't working properly (I think) and there were only two or three people following me, being 2 of them bots. The other one also follows me through RSS so I hope I don't affect them very much.

Today and tomorrow I'll also proceed to make some internal upgrades and decouple the webmentions from the static generation process of this website, inspired by [Jan-Luka's post](https://jlelse.blog/micro/2020/07/webmentiond/). I need to upgrade to Caddy 2 as well as some other changes I might share later.