---
description: |
  Data is a topic I've been wanting to write about for quite some time. Do we own our data? What should we do about it?
publishDate: "2019-12-28T15:30:00.000Z"
tags:
- indieweb
- url
- meta
title: URL Structure
---

I’m now working on making my website more IndieWeb friendly, which was triggered by my searches after writing my last post about [owning our own data](/articles/2019/12/own-your-data/). It has been… harder than I though. But in a positive way!

There are so many, but so many concepts in the IndieWeb world that I could never imagine before, but first: I need to support multiple data types because this website will now on hold more than just long articles like this one. We will support notes, bookmarks and articles for now.

<!--more-->

I’m using the static website generator [Hugo](https://gohugo.io) which basically allows me to customise my website the way I want, but they have so many concepts: sections, types, taxonomies, categories, tags…

So, first of all, let’s think about the URL structure I want to have and then how to do it. My requirements were (simply copied from my notes):

* Must be human readable
* Must be easy to implement with Hugo
* Must be possible to separate by category

Before starting this, this is what I had:

```plaintext
hacdias.com/YYYY/MM/post-slug
hacdias.com/blog
```

After jotting down 4 different ideas, I decided to choose none of them and pick this one:

```plaintext
hacdias.com/YYYY/MM/DD/XX/post-slug     <— article
hacdias.com/YYYY/MM/DD/XX/              <- notes
hacdias.com/YYYY/MM/DD/XX/              <- bookmarks
```

And this URL structure is reflected directly into the file system. So, we have a directory per day and inside it we have each post. The `XX` is a numbering we give to each post, so the first one will be `01` , the second one will be `02`.

Hmm… Do I like it?

I could like it better honestly, they seem they can get too big, but this way they’re also really easy to manage on the file system and everything just works.

Why not leaving them per month as I had before? Since I’m going to have more post types, I’ll likely post more than once per day and if I posted more than 99 times per month, I would get links with different sizes, which I didn’t want (yes… I know it might be stupid…).

Now: how did I achieve this with Hugo?

It was quite easy! I just structure the directories inside content like the URLs and then gave each file a category (blog for articles, notes and bookmarks). Why not create types? True, it would’ve been much easier and I could simply create a layout file for each one, but it would be harder to create listing directories. But… is it worth the hassle?