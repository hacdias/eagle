---
publishDate: "2020-01-01T23:30:00.000Z"
syndication:
- https://twitter.com/hacdias/status/1212517049035038720
tags:
- indiweb
- micropub
- ownyourowndata
title: Adding support for Micropub endpoint
---

Recently, I have talked about [restructuring the URLs](/articles/2019/12/url-structure/) of my website and [adding IndieAuth](/articles/2019/12/using-indieauth/) so I could use my domain as my main  online identity to login into services. Along those lines, I came across [Micropub](https://www.w3.org/TR/micropub/). In their own words:

> The Micropub protocol is used to create, update and delete posts on one’s own domain using third-party clients.   

So it’s basically a simple common protocol that could let any website get updated using an arbitrary CMS (Content Management Software) or application that supports it. I really enjoyed the spec and there’s [suggestions and issues](https://github.com/w3c/Micropub/issues) being worked on.

<!--more-->

What I’ve been trying to do is to implement this on my own website. There aren’t  many open source [server implementations](https://indieweb.org/Micropub/Servers) unfortunately. However, I understand the reason: most of them are highly specific for their author’s website and so is the one I’m building.

I’ve been building it using [express.js](https://expressjs.com/) and I’ve already published two different nom modules:

* [@hacdias/micropub-parser](https://www.npmjs.com/package/@hacdias/micropub-parser)
* [@hacdias/indieauth-middleware](https://www.npmjs.com/package/@hacdias/indieauth-middleware)

Apart from that, I’m mostly calling APIs directly: I’m relying a lot on some of [@aaronpk’s](https://aaronparecki.com/) modules, such as [Compass](https://compass.p3k.io) and [XRay](https://xray.p3k.io/) . Those are, btw, awesome modules and you should try them out if you’re interested in this subject.

The current status is that I pass almost all the tests provided by [micropub.rocks](https://micropub.rocks) even though I’m not satisfied with the solution I’ve come across. Why?

First of all, if you’ve been following me for more time, you might’ve came across an [old blog post](/articles/2015/08/farewell-wordpress-hello-hugo/) talking about how I am using [Hugo](https://gohugo.io) to generate my website. But that is not the only “problem” of this: I’m also using IPFS to store my website for the folks using it. So there’s quite a few constraints.

## Problems
### Build Speed and Complexity
It needs to build fast so every time I post something new, I don’t need to wait a few seconds. Hugo is known for being fast, but I’ve about ~950 posts on this website right now divided between replies, notes, likes and articles that I’ve imported.

It takes ~1s to build on my machine and ~3s on my server which is definitely not ideal. I don’t know why this is taking so much time.

In addition, the complexity of my themes is really big and that might be caused by the not-so-high flexibility of Hugo and Go templates. There are template languages that are more flexible and static website generators that allow plugins, such as Jekyll — not an option since I need speed — or 11ty.

### Static
All links must be relative and the site must be 100% static. Any dynamic stuff must be done on the client-side so it plays well with IPFS and other distributed technologies that don’t support server-side rendering.

Personally, I would love to have a better search engine working for myself, to find bookmarks, to find old replies. Tags are definitely not enough for this and search engines take quite a few time to get updated with all the new content. And… do I want all of the content on search engines anyways?

## Solutions and Thoughts
Obviously I’ve been thinking a bit about this and some conclusions were drawn to my mind. So let me “dump” them here in a rational way.

### Build Speed
I thought about just moving out of a static website generator and have some more dynamic website. With that, I would be able to have more powerful queries and basically solve the problems from the “Static” point I mentioned above.

Not to mention that if the pages were built on the fly with server-side generation, I would not require full builds every time. But… is it worth it? That way I couldn’t cache it on IPFS. What if I cached the website once per day and updated my static site on IPFS? These are just some questions I’m throwing here.

Another interesting option here would be to continuously run `hugo —watch`. That way Hugo would be watching for updates and only the required pages would be rebuilt. Should I do that?

### Static
Two solutions: one would be what I said on the previous point and just make the website server-side rendered. It would open an immense sea of opportunities and things I could do.

Or, another interesting way of putting things, build JSON indexes that could be used by client-side server engines such as [Lunr](https://lunrjs.com/). However, as the website would become bigger and bigger, I am pretty sure the indexes would become bigger and bigger and the load times slower and slower.

- - - -

Well, these are just some problems I need to solve before continuing and I would personally love your take on them. What should I do in your opinion? Please [lemme know](https://twitter.com/hacdias/status/1212517049035038720)!