---
description: |
  It's now time to own my own reading log. I started by creating a reading logs page and supporting all the IndieWeb-related specs for this.
publishDate: "2020-01-29T23:50:00.000Z"
tags:
- meta
- reading
- books
- indieweb
title: Owning my reading log
---

As [Tom](https://macwright.org/2017/12/11/indieweb-reading.html) once said, it is now time to own my own reading log. Why? Despite all the reasons mentioned on Tom's post, I also got bored of Goodreads and I ended up not using it as much as I should have.

With university, work and… life… I stop reading as much as I did before. But it's now time to get back to some reading. Even if it's not that much, I need to read something. I must do it.

<!--more-->

At my [reads](/books/), you can visualize my reading logs: basically, it's just a big list of 'I want to read X', 'I finished reading Y' or 'I am now reading Z'. I will be primarily using the [indiebookclub](https://indiebookclub.biz/) service for now to create this kind of posts.

On my [books](/books/), you can find my bookshelf of the books I have read in the past. Of course, you won't be able to find _all_ the books I've read. That list's missing at least three hundred comic books I have in my hometown. I know, that's a lot.

The bookshelf page is based on the logs from my reading logs. I think I will also add a want to read shelf and currently reading shelf. However, that will be a little challenging for me. I know how to do it. However, I don't know if that's the best way to do it.

Right now, I'm just filtering the books by the reading status. But then, once I start using this, I will have the same book on multiple statuses. How do I know if I'm in the last status? I'm thinking about the simplest solution possible:

When adding a new status for a book, change the previous status and add a tag such as #noshelf and then, when building the page, I would know which ones I should add or not to the page.

In other thoughts, I'm storing the files as much as microformats-like possible. However, I'm getting strongly cumbersome files. Just look at this:

```yaml
categories:
  - reads
date: 2020-01-29T23:19:58.372Z
properties:
  category: &ref_0
    - story
    - classic
  read-of:
    - properties:
        author:
          - Lewis Carroll
        name:
          - Alice's Adventures in Wonderland and Through the Looking Glass
        uid:
          - “isbn:9781853260025”
      type:
        - h-cite
  read-status:
    - to-read
tags: *ref_0
```

I am making an effort to try to save everything the closest to the microformats  spec as possible. And that increases the templates complexity and makes the file harder to read.

There's a spec for a minimal version of microformats called [jf2](http://microformats.org/wiki/jf2) which looks promising. Perhaps I'll try  doing that. The previous example could be compressed to something like:

```yaml
---
categories:
  - reads
date: 2020-01-29T23:19:58.372Z
properties:
  read-of:
  	properties:
     author: Lewis Carroll
     name: Alice's Adventures in Wonderland and Through the Looking Glass
     uid: "isbn:9781853260025"
    type: h-cite
  read-status: to-read
tags:
  - story
  - classic
```

This would be easier for the templates. But then, I would need to make more transformations when receiving, updating and generating the microformats on the micropub endpoint. I could also get rid of `properties` altogether and just add that level during transformations.

I'll add this as a ToDo and, if I have time, I'll tackle that. For now, I have a nice read logs page and a working bookshelf!