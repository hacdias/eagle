---
publishDate: "2020-02-16T11:06:57.887Z"
tags:
- meta
- organization
---

So I just made a few changes to my website and I hope it didn't break anything like feeds and such. Here's a small changelog of the changes:

- Stopped using Hugo [categories](https://gohugo.io/categories/) for post types (replies, notes, articles, etc) and started using sections, i.e., I know put a note under the `/note` path. So this also changed the URLs, hopefully for better and now it's easier to restrict access or remove something if I want.
- I added about ~2000 redirect rules. Does anyone know if Caddy allows me to import the redirect rules from another file? My Caddyfile is getting huge.
- Started using [ `partialCached`](https://gohugo.io/functions/partialcached/) in some places which improved the build time a tiny bit.
- Moved the Articles page from `/blog` to `/articles` which I already wanted to do for a while.
- Added a [contact](/contact/) page.
- Updated the [more](/more/) page with more links!

And... that's it I think. I'd also love to use this website as a "knowledge base" so I'll probably create a section for that later. I always want to organize the knowledge I get somehow but I just have tons of files from university and other stuff laying around without any organization. I really loved this [braindump](https://braindump.jethro.dev/) from [Jethro](https://www.jethro.dev/).