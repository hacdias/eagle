---
description: |
  It's now time to own my own watch log. I use Trakt to keep up with the series and movies I'm watching and now I'm going to PESOS to my website!
publishDate: "2020-02-13T20:50:00.000Z"
tags:
- meta
- movies
- series
- ownyourowndata
- indieweb
title: OwnYourTrakt
---

For quite some time, I have been getting more and more into the IndieWeb world and trying to own my own data. I have started publishing more to my website and using it as a place to store most of my public data, i.e., data I already published on other social media and platforms.

It now holds my web interactions, such as [replies](/micro/), [likes](/micro/) and ~~reposts~~, as well as my [reading log](/articles/2020/01/owning-reading-log/). Since the beginning, I also wanted to this website as a place to store my watch logs. With watch I mean watching movies and TV series.

<!--more-->

Since about 2017 I’ve been using [Trakt](https://trakt.tv) to add track the movies and TV series I’ve been watching.  Since Trakt proves itself valuable, I decided to follow the [PESOS](https://indieweb.org/PESOS) strategy where I start to publish on a third party service, in this case Trakt, and then publish to my own website.

Inspired by [@aaronpk’s](http://aaronparecki.com/)  [OwnYourSwarm](https://ownyourswarm.p3k.io/) project, which allows to publish your Swarm activity on your own website, I built a tool called OwnYourTrakt which is, of course, [open source](https://github.com/hacdias/ownyourtrakt).

For now, I’m not providing a hosted version of this service and please remember that, if you use it, use at your own risk. I also encourage you to help me improve it and eventually come up with a hosted version.

 To create this, I started by looking at Trakt’s [API docs](https://trakt.docs.apiary.io/) which are quite powerful compared to what I was expecting from them. They use [OAuth 2](https://oauth.net/2/) and it was my first time implementing the OAuth 2 login logic on a web service and I think I learned quite a few things.

In a really simplified way, this is what OwnYourTrakt does:

1. Connects to your website and gets an access token for your micropub entrypoint.
2. Connects to Trakt.
3. Every 30 minutes checks for history updates on Trakt. Note that the history uses the actual date when the movie/episode was watched. Thus, if you now add a movie as watched a week ago, it might not be fetched.
4. If there’s a new item,  a micropub request is made to your entrypoint.

A simple guide on how to set up an instance of OwnYourTrakt and what I include on the requests is described on the [readme](https://github.com/hacdias/ownyourtrakt#own-your-trakt).

There's no official way to render watches as microformats so I just decided to go with something along these lines:

```html
<p class='copy content e-content e-entry'>
  Watched <a target="_blank" rel="noopener noreferrer" href="$url">$title</a>
</p>
```

~~You can see my watch log at /watches!~~ Please check [this post](/micro/2020/02/removed-watches-checkins/) to know why I changed my mind.