---
publishDate: "2020-02-01T14:52:09.397Z"
tags:
- meta
- cdn
- micropub
- media
---

I just set up a media endpoint based on [BunnyCDN](https://bunnycdn.com), inspired by @jlelse's [post](https://jlelse.blog/micro/2020/01/2020-01-01-frviz/). So far, it's working really well.

For now, I'm not actually using it to post many of the images of the website, even though I could. However, I'm using it to store the webmentions author's photos. They were being served directly by [webmention.io](https://webmention.io/) but I think it's better to serve them myself.

The media endpoint works well: it receives an object and stores it on BunnyCDN. However, I want to add some customization options such as resizings and compressions for images through query parameters, as well as some default ones so I don't need to always specify them.