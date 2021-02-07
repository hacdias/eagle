---
publishDate: "2020-10-14T10:09:25Z"
replyTo: https://jan.boddez.net/notes/046185c328
tags:
- meta
- activitypub
---

The thing is: I've done this in the past. I just tried with a random Mastodon instance and the error is precisely the HTTP signature. Either I'm not signing correctly, or I'm not proving the public key correctly. But the fact that I did this in the past and it's still not working, it's what's annoying me the most. Also, I'm doing this in Go and I'm basing on [@jlelse's](https://jlelse.blog) implementation and his ActivityPub Accept is working. And my code literally looks the same. If I don't figure it out in the next hour, I'll leave it for another day.