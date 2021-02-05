---
description: I finally learned how to say goodbye to WordPress. Not forever. Nothing lasts forever. But for my blog. I'm using Hugo. A very simple static website generator."
publishDate: "2015-08-12T11:30:00.000Z"
tags:
- meta
title: Farewell, WordPress! Hello, Hugo
---

Firstly, a little bit of history - when I begun this blog, I've already been using WordPress on Pplware for a while. So, I tought: I know it, it's simple, it's easy to use, I'm going to use it on my personal blog.

After deciding I was going to use WordPress, I bought a domain on [GoDaddy][1] (which I recommend) and paid for hosting on [PTServidor][2]. With the blog set up, I started writing posts in portuguese. I did it for some time.

<!--more-->

Less than an year afterwards, I moved my blog to [DigitalOcean][3] (which I recommend too). I had 100$ from GitHub Education for free to use on DigitalOcean and I give it a try. It was a nice and useful experience: I learned how to set up a server, apache and some of other things in a production environment.

I was using a cache plugin to speed up the blog and also using the Yoast SEO plugin. I think both are really useful and I recommend them if you're using WordPress.

{{< figure src="seo-what.jpg" alt="What's SEO?" caption="What's SEO?" >}}

Two months later - I think it was two months, I'm not sure - I decided to use Jekyll on [GitHub Pages][4]. Jekyll is a static website generator, i.e., it converts some source files to static and plain HTML. After fighting a lot with Ruby on Windows, I moved my blog from WordPress to Jekyll.

Everything was running fine until I formatted my computer. And then I tought: no, I'm not gonna install Ruby again, not on Windows, I'm not enter the hell. I searched for more static site generators and I found [Hugo][5].

Hugo is a really nice and easy to use static website generator, built using Go (one of the languages I admire), that have standalone executables. It **doesn't have dependencies**. It's simple, easy. Why not?

{{< figure src="https://cdn.hacdias.com/uploads/2015-08-writing-post.jpeg" alt="Writing a post" caption="Writing a post" >}}

I moved everything to this new system and created a new template (the black one before the current one). It's very simple to [create themes](http://gohugo.io/themes/overview/) for Hugo. All of my blog's code is on [```henriquedias-source```][6] at GitHub. Then, I just have to deploy it to [```hacdias.github.io```][7] repository so I can use GitHub Pages hosting which is free.

I also configured [CloudFlare][8] and my website is very fast now. It's delivered by their CDN and I'm using SSL. **I defend that every website should be using HTTPS**.

{{< figure src="https://cdn.hacdias.com/uploads/2015-08-speed-insights.jpeg" alt="My blog score's on Speed Insights" caption="My blog score's on Speed Insights" >}}

Concluding, I'm saving money because I'm only paying the domain. I'm using a easy to use system (of course Hugo is not for everyone, but it's simple). My blog is faster than ever. Google Page Speed Insights gives me a very high score. I'm very satisfied with Hugo.

If you're confortable with Markdown, HTML, CSS and JS, I **recommend** you Hugo with GitHub Pages.

[1]: https://godaddy.com/
[2]: https://www.ptservidor.pt/
[3]: https://www.digitalocean.com/
[4]: https://pages.github.com/
[5]: http://gohugo.io/
[6]: https://github.com/hacdias/hacdias.com
[7]: https://github.com/hacdias/hacdias.com
[8]: https://www.cloudflare.com/