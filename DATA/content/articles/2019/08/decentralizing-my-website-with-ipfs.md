---
description: How does my website work with IPFS? Where does the content go? What happens when my website is accessed through HTTP? Here's a tale that explains of what happens.
emoji: "\U0001F310"
publishDate: "2019-08-07T18:45:00.000Z"
tags:
- ipfs
- dns
- dnslink
- meta
- decentralization
title: Decentralizing my website with IPFS
---

It was about two years ago, in 2017, that I got to know IPFS through [David Dias][0]. This is not the first time I talk about IPFS here, in my website, but here's a brief description of what it is: IPFS stands for InterPlanetary File System and it is a peer-to-peer, content-addressed and decentralized protocol.

<!--more-->

By decentralizing my website, I want you to fetch it from the nearest source you can. If your accessing this through HTTP, that might not be what's really happening (_kinda_, will explain later). But if you already know IPFS and you're using [IPFS Companion][1], changes might be that you're loading this through a decentralized network.

First of all, I use a static website generator called [Hugo][2] which does its job immensely well and I give my props to its creator and maintainers. However, we're not here to talk about static website generators, but about how does my website work and how you can do the same about yours.

Two simple commands are all that is required to update my website to the InterPlanetary network. Actually, I just need to push it to the git repository because I have [CircleCI setup][3] with the required commands. Anyways, here are the commands:

```bash
hugo --gc --minify
npx ipfs-deploy public -p pinata -d cloudflare
```

Simple, right? Let's break it down a bit to see what's going on here. The first command is the website generator, which outputs to a directory called `public`. Nothing much to say about that. Only that it contains all the files of my website. Then, we run `ipfs-deploy` and the magic happens!

[`ipfs-deploy`][4] is a tool built by [@agentofuser][5] that does the following: takes a certain directory - in this case `public` - and adds it to as many pinning services as we want and updates the DNSLink on the DNS provider we defined.

**Wait, what? Pinning services?**

So, a pinning service is an online service that runs an IPFS daemon - and usually a gateway too - and that assures you that your content does not go away from the network. And now you tell me: doesn't that invalidate the point of being decentralized and needing no computer to have the website online?

Well, you could run a local IPFS daemon - [as you should be doing!][6] - and add your content through your own node on your own computer. But, if no one else has the contents of your website, then it just won't work unless you have your computer connected to the Internet 24 hours.

**What about DNSLink?**

DNSLink is a TXT record on your domain that points to a certain IPFS hash. Let's say your website is available at `example.com` and the hash of the content of your website is `QmHash`, then, you should've a TXT record at `_dnslink.example.com` with the value:

```txt
dnslink=/ipfs/QmHash
```

DNSLink allows IPFS to know what the content of your website is just by looking at its domain. Certainly, you wouldn't to give people hashes to access your website.

Let's now talk a bit about each service!

## Pinata

Pinata is a freemium service that allows you to pin up until 1 GB of data for free. You need to sign up at [their website][7] and then you'll be able to get your api key and secret api key. You'll need to set [two environment variables][8] with those values in order for `ipfs-deploy` to work.

## A tale of two oranges

I'm using Cloudflare as my go-to DNS provider. Since this year, Cloudflare has had [their own IPFS gateway][9] running. What you can do is: if your domain is `example.com`, set a CNAME record at the root to `cloudflare-ipfs.com`. That will mean that your website will be served by Cloudflare's gateway.

Pinata and Cloudflare maintain a persistent connection so the content discovery will certainly be fast so there won't be any problems when updating the website.

**Uh, I already used Cloudflare and now 'Always on HTTPS' is not working anymore... What did I do wrong?**

Nothing, you did nothing wrong. Right now, since your website is pointing to another Cloudflare website (their IPFS gateway), the rules applied at `cloudflare-ipfs.com` will ditacte how your website goes. And, for many different reasons, the IPFS gateway has 'Always use HTTPS` disabled.

But this will be solved! Cloudflare is working on a solution called [Orange to Orange, or O2O][10], which will allow you to override settings even if your website DNS points at another website served by them!

As for `ipfs-deploy`, you'll also need to set [some environment variables][11] to get it configured in order to be able to automagically update your DNSLink at Cloudflare.

Then, just run:

```bash
ipfs-deploy -p pinata -d cloudflare
```

And you're website will be pinned to Pinata and your DNSLink at Cloudflare will be updated.

**But that's a lot to make a website decentralized...**

It's a one time process: you set it up and leave it running. As mentioned before, I'm using CircleCI to build my website, pin it and update the DNSLink every time I commit to the master branch. You can take a look at the [setup][3].

**Is it really decentralized?**

Yes! DNS is decentralized, Cloudflare is a distributed CDN. Your website points at Cloudflare IPFS gateway, which is a cached IPFS gateway served by the distributed Cloudflare network. And Cloudflare also needs to get the content from Pinata - or your computer if you don't want to pin it to an external service!

**What can you do to improve?**

Start by installing [IPFS Desktop][12], which installs you an IPFS daemon and leaves it running all the time, giving your computer superpowers. Then, install [IPFS Companion][13], the browser extension that will power up your browser! If you do that, next time you open my website (and many others!), you'll be redirected to your own gateway, provided by your own IPFS node, and it will fetch the website from wherever it finds the content!

Happy decentralization!

[0]: http://daviddias.me
[1]: https://github.com/ipfs-shipyard/ipfs-companion#install
[2]: https://gohugo.io/
[3]: https://github.com/hacdias/hacdias.com/blob/master/.circleci/config.yml
[4]: https://github.com/ipfs-shipyard/ipfs-deploy
[5]: https://github.com/agentofuser
[6]: https://github.com/ipfs-shipyard/ipfs-desktop
[7]: https://pinata.cloud
[8]: https://github.com/ipfs-shipyard/ipfs-deploy#pinata
[9]: https://www.cloudflare.com/distributed-web-gateway/
[10]: https://blog.cloudflare.com/continuing-to-improve-our-ipfs-gateway/
[11]: https://github.com/ipfs-shipyard/ipfs-deploy#cloudflare
[12]: https://github.com/ipfs-shipyard/ipfs-desktop#install
[13]: https://github.com/ipfs-shipyard/ipfs-companion