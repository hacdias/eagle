---
description: |
  How I built a cloner for xkcd and keep the clone up to date on IPFS.
publishDate: "2019-07-05T16:20:00.000Z"
tags:
- ipfs
- clone
- code
- decentralization
title: Mirroring xkcd comics to IPFS
---

As many of you probably know, xkcd is a web comic created in 2005 by Randall Munroe. Its tagline is "A webcomic of romance, sarcasm, math, and language". You can read more about the web comic itself on their [website](https://xkcd.com/about/).

As part of wanting to contribute to [IPFS Archives](https://github.com/ipfs/archives), I decided to look at the issues and find the most interesting one for me, and I did: xkcd! There were some clones already built, but all of them were out of date and weren't automatically updated.

<!--more-->

So I started by building a small tool that would fetch all the comics and their info. I called it... as you might image... [`xkcd-clone`](https://github.com/hacdias/xkcd-clone). It doesn't actually clone the whole website, but only the comics and their metadata.

{{< figure src="2165-thumb-social.jpg" caption="Comic 2165" >}}

The tool can either download everything from scratch or just update an old version of a clone. You can use it like this:

```bash
npx xkcd-clone -d <directory> [--empty]
```

By running it, you'll get a directory with the following structure:

```text
xkcd
 | 0001
   | barrel_cropped_(1).jpg
   | image.jpg
   | index.html
   | info.json
 | ...
 | 2171
 | index.html
 | tachyons.css
 | tachyons-columns.css
```

Where `barrel_cropped_(1).jpg` is exactly the same image as `image.jpg` but preserving the original filename. Since we'll store it in IPFS right away, it won't take more space for it because they point to the same place.

Then, I set up a "placebo" repository on GitHub that runs on CircleCI everyday. It's [hacdias/xkcd.hacdias.com](https://github.com/hacdias/xkcd.hacdias.com) and basically this is the procedure that updated the clone everyday:

1. Gets the cached output from the last run (to save resources).
2. Fetch the latest xkcd comics using [xkcd-clone](https://github.com/hacdias/xkcd-clone).
3. Pins it to IPFS Cluster.
4. Updates the DNSLink of xkcd.hacdias.com to the latest hash.

Now, the clone is published at [`/ipns/xkcd.hacdias.com`](http://dweb.link/ipns/xkcd.hacdias.com/). You can check out the latest hash by running the following command on your terminal:

```bash
$ ipfs name resolve /ipns/hacdias.com
/ipfs/Qma24VwKNSJXFcueh5wnRpnncvnK6HRWG1P5BT7fnWbPWM
```

You can even pin it directly by running:

```bash
$ ipfs pin add /ipns/xkcd.hacdias.com
```

There are still, of course, some points that won't be achieved soon such as "what should we do with dynamic comics?". Perhaps we could make a costum procedure for each one, perhaps not.

If you're interested in contributing, you're more than welcome to look out at the [code](https://github.com/hacdias/xkcd-clone) and improve it even more! PRs welcome as usual!