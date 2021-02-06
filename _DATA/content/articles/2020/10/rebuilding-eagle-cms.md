---
publishDate: "2020-10-12T09:00:00.000+02:00"
tags:
- meta
- eagle
- activitypub
- mondayletter
title: Rebuilding Eagle, my website's CMS
---

My website is growing, not in terms of views, because I'm not aware of that. Maybe I should add some non-intrusive analytics. Anyways, it's growing in terms of functionality. However, since I'm using a static website generator, it makes it complicated to add some interactive functionalities.

One of those features is [ActivityPub](https://www.w3.org/TR/activitypub/). ActivityPub is, in the words of the spec:

> The ActivityPub protocol is a decentralized social networking protocol based upon the ActivityStreams 2.0 data format. It provides a client to server API for creating, updating and deleting content, as well as a federated server to server API for delivering notifications and content.

I have [tried][apub1] to build complete [support][apub2] for ActivityPub in the website before, but it didn't succeed and I ended up just removing it instead of fixing it. In the meanwhile, I thought about creating a Pleroma instance, but what's the point? I already have a section for [micro](/micro/) posts. Thus, the website itself should be able to interact with the "outside world". By outside world, I mean all the other ActivityPub servers, such as all Pleroma and Mastodon instances.

Another feature I want to support is search. Why? "Can't you just use Google's site search?", you ask. Yes, I could, but it wouldn't be as great, nor as updated. Also, by implementing my custom search for this website, I could use some engine that allowed for fancy filters.

Right now, my website already supports Micropub and Webmentions, so it's kind of interactive already, despite being static. It is simply rebuilt every time. Thanks to Hugo, the static website generator I use, that is quite a fast process (2 seconds ±).

So my idea is not to create a full-blown CMS, just like [@jlelse][goblog] is doing, but to create a wrapper around Hugo. Well, if we consider a CMS to be literally a 'Content Management System', then that's what I want to build, but the generation of pages will still be done by Hugo.

I can see two main advantages: 1. I can focus on the functionality I want to build without having to worry about template parsing and all that - just keep it as it is! - and; 2. the static result can still be hosted in IPFS or any other decentralized shared file system.

This is [not][cms1] [the][cms2] [first][cms3] time I [think][dynamic] about building my own CMS though. I had the same thoughts during quarantine, a few months ago, but ended up not implementing anything. Right now, I managed to move my [old API](https://github.com/hacdias/eagle-js) from Node.js to [Go](https://github.com/hacdias/eagle) and I'm quite happy with the result.

I managed to migrate the Micropub, Webmentions and all other services I had to the new Go system. And... I just hope everything is working. I tested it thoroughly on a testing domain and it seemed fine and fixed all te bugs I could find. I'm pretty sure more will show up soon, but that's something for another day!

With this change, I managed to implement [search](/search/)! I am using a search engine called [MeiliSearch](https://meilisearch.com/), which is built in Rust, and blazing fast. Just try it out! I didn't implement the "update results while typing thing" so you need to press the button. It works! And it is fast!

In the future, I want to fully support ActivityPub though. I still need to implement my Inbox to make it fully work! Well... I know this was a different post than the other days, but I hope you enjoyed it!

[apub1]: /micro/2020/03/n1/
[apub2]: /micro/2020/03/i-now-have-activitypub/
[goblog]: https://git.jlel.se/jlelse/GoBlog
[arewewebrust]: https://www.arewewebyet.org/
[cms1]: /micro/2020/01/on-building-my/
[cms2]: /micro/2020/07/how-to-publish-notes/
[cms3]: /micro/2020/01/n1/  
[dynamic]: /articles/2020/01/dynamic-static/