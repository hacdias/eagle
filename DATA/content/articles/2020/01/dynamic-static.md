---
publishDate: "2020-01-02T23:40:00.000Z"
syndication:
- https://twitter.com/hacdias/status/1212881805096996864
tags:
- meta
- indieweb
- micropub
- dogfood
title: Build a dynamic website… or not?
---

After digressing a bit about [building a Micropub](/articles/2020/01/building-micropub/) endpoint for my website, I’ve been thinking about the next steps: if I should keep Hugo or move to some other system.

I opened a topic on [Hugo’s discourse](https://discourse.gohugo.io/t/use-hugo-watch-in-production/22599/14) to see to which point it would be reliable to use the watch mode of Hugo in production and it seems, as I was expecting, that it wasn’t made for that and there is no way of making incremental builds as of right now. And I don’t blame Hugo developers — I understand that’s quite a big issue and it needs to be a really thoughtful process.

<!--more-->

In addition to that, the biggest issue I might be having is actually with my templates. If I remove them, I cut the build time from ~1s to ~0.5ms, which is half the time!

My first attempt will be to improve the template layouts and make them faster than they are right now to see the results. However, I don’t think that will be the solution I will end up picking!

{{< figure src="comic-social.jpg" class="invert" >}}

I’m the person who always takes the hardest path and the most difficult to accomplish for whatever I want to do and I’m feeling that probably I shouldn’t in this case as I’m right this. Let me jot down some thoughts.

## What do I want to do that is more difficult to do with a static website and why?

First and foremost, adding content to a dynamic website that is rendered on the fly would be faster because I wouldn’t need to rebuild the **entire** website every time I change something. However, using server-side rendering I would also have slower response times from the end-user perspective.  But…  would a few milliseconds be that harmful? I doubt it. Also, there’s caches.

Secondly, I would love to have a powerful way of searching stuff on my website. Yes, I could build static indexes that’d get updated every time I rebuilt the website. Problem? Full text search would require to download the content of **every single post** on my website so the search would work. Is it worth it? I don’t think so. For this, an indexed database could be used.

Finally, experimentation! Building my custom server “thingy” could actually be a good way of dogfooding.  I would be able to, perhaps, just saying, create some interface to check private Webmentions or even have private posts or bookmarks. I don’t know, just throwing some ideas.

## If I moved out of Hugo, which technologies would I use?

So, I’ve been thinking quite a bit about this. Firstly, I must say that I don’t know yet if I should divide the website itself from the API. Should it be the same application? Or different ones?

About the language: Node.js is “new” and shiny and I’ve been using it **a lot** on my past projects. But I don’t think I would use it here even though I have published [two packages](/micro/2019/12/two-packages/) for helping building Micropub applications a few days ago. “Why?” you ask.

PHP is old, but it has had really impactful improvements since the last time I tried it and there are a lot of PHP packages already built by other IndieWeb lovers such as [Aaron](https://aaronparecki.com/) and [Tantek](http://tantek.com/) I could leverage. Also, as far as I remember, templating and outputting stuff with PHP is really… simple.

Fun fact: PHP was my first programming language. After weeks of research, I picked C# and I bought a book. But for some reason that I can't recall, I never read it and jumped into PHP.

This would also be an opportunity to try some PHP framework (Laravel, Lumen, ?) to help me build the website without much hassle, but should I? Do I need it? Will it help me a lot or just a bit?

The files would be structured in a really similar way to p3k’s. They’re actually already almost like that on my Hugo version:

```text
YYYY/
  /MM/
    /DD/
      /XX/
        /index.md
        /public/
               /public_file_1.jpg
somepage/
        /index.md
```

Where `index.md` would have a YAML metadata bit and the rest would be markdown. I also found a markdown library called [parsedown](https://parsedown.org) which seems quite fast!

Then, I would use some sort of database indexing for search and listing pages. However, I would not be storing my [main posts copy](https://indieweb.org/database-antipattern) on the database! The database would be just an indexing and querying tool for speed and ease of access and it must be easily rebuilt from the file system contents.

## What about IPFS?

Here’s the hardest question! I thought about generating a static copy and updating my DNSlink every day. This “copy” would not have search, for example, and other features that’d require server-side rendering.

## Truth?

Everything I said before is true although I also want to say this: truth be said, I always wanted to build some sort of system like this to use on my day to day and have it running and actually maintain it.

Since the very beginning (I bought henriquedias.com more than 5 years ago already!), my website has been almost always static. For a short time I used WordPress, but it didn’t take long for me to drop it. I want a custom solution that uses open protocols, but also allows me for flexibility. Does that sound sensible?