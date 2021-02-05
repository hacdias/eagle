---
description: |
  After some considering, I think it is time to build my own CMS.
publishDate: "2021-02-04T22:22:00.000+01:00"
tags:
- cms
- meta
title: Is it time to build my own CMS?
---

Today I have been thinking about the possibility of writing my own CMS. A very simple CMS. I even [hinted](/micro/2021/02/am-i-building-a-cmd/) that on a small micro post today. This is not a new idea. I'm not going to look at the previous posts where I talked about this, but it's definitely at least 2 or 3.

"But why?", you ask. Well, the answer is more complicated than I would like it to be. There is a number of features that I want to implement that will just make my website inherently more dynamic. And as @jlelse once [said](https://jlelse.blog/thoughts/2020/07/own-cms) "it’s almost questionable why I use a static page generator at all".

## Let's talk about features!

Right now, my ["CMS"](/articles/2020/10/rebuilding-eagle-cms/) is, in fact, a simple wrapper around Hugo with a few bells and whistles. Also, besides wanting to have a simple CMS, I also want to build a simple CLI that allows me to locally search, list, add and edit files, as if it was my own notebook. For that, I am taking some inspirations from [nb](https://github.com/xwmx/nb), which the authors perfectly describe as:

> CLI plain-text note-taking, bookmarking, and archiving with encryption, filtering and search, Git-backed versioning and syncing, Pandoc-backed conversion, and more in a single portable script.

For the sake of clarity, here are the features/things I want to implement/change on my website:

1. **Dashboard** that allows me to...
	- create, update and delete posts. With this, I would remove my Micropub endpoint. Right now, it is a bit of an hassle to transform between the Micropub format to the internal format and vice-versa. More than that: I don't even support all the features I would like to.
	- approve comments dynamically.
2. **Webmentions** would still be available but there would be a native comments box on the bottom of each page, where everyone could leave their own comment without relying on [commentpara.de](commentpara.de).
3. Improve the current **search** functionality. Currently, I support full text search but it is a bit cumbersome and hidden.
4. Revive the **bookmarks** section!
5. Stop relying on GoodReads for my **reading** section and make every reading activity an actual post.
6. As mentioned before, a **CLI** that would allow me to manage this things locally. Besides, this CLI would also ensure that the changes are automatically committed and pushed.

Those are the main features and changes to the functionality of the website. In addition, there's some inner workings that I would like to change. I feel that the current files hierarchy just makes everything complicated for me to access: all posts are a directory with an index, a webmentions JSON file, plus some other files that I might need.

I want to separate the written content from the images and other special data. With a new CMS, I would have more flexibility on how to name my files and I'd just move all my media to BunnyCDN.

## Should I Go or should I Rust?

That's a very good question. I have **never** used Rust in my life, but everyone talks about it! Is it _that_ good? I know I would code faster in Go because I know it. But should I try going for the shiny "new" thing and learn something new?

So you know, these are my current inspirations:

- [https://github.com/hacdias/eagle](https://github.com/xwmx/nb)
- [https://git.jlel.se/jlelse/GoBlog](https://github.com/xwmx/nb)
- [https://github.com/Xe/site](https://github.com/xwmx/nb)
- [https://github.com/xwmx/nb](https://github.com/xwmx/nb)

Only `Xe/site` is in Rust. Ugh. I don't really know. I tried to setup a small CLI in Rust and it too so much time to compile! Besides, I'm afraid of going deeper into Rust and then regretting.

What would you do? What do you think? I will definitely appreciate your opinion on this!