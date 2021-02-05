---
description: |
  Here I describe my small journey so far with Notable and how I sync my website's Knowledge Base it with.
emoji: "\U0001F4D3"
publishDate: "2020-02-29T21:37:00.000Z"
tags:
- meta
- notes
title: How do I generate my knowledge base
---

For quite a few time, I used [Bear](https://bear.app/) as my go-to Notes application for two reasons: it was simple to use and the syntax was quite similar to markdown. However, it is not markdown and it does not support some things that'd like to see on such software: diagrams, mathematics, wiki-like links, etc.

After searching for a bit I found out an app called [Notable](https://notable.md/). In addition to having all the features I wanted from Bear, it is also storage independent, meaning everything is stored as markdown plain-files that I can version control with Git or some similar software.

<!--more-->

I only have one small complain about the project and it is the fact that it is not open source **anymore**. It is also free for now, but it might change. Please take a look at [this issue](https://github.com/notable/notable/issues/432) where the author explains their decision.

Nevertheless, it is a real nice piece of software that has been serving me well. But that is not what this post is about: I'm here to explain how I generate my knowledge base page from my Notable notes.

It is quite simple actually. First of all, I have a git repository - hosted on GitHub - where all my notes are. Every 15 minutes, I have a script that runs on my machine and pushes if there are any changes. It looks like this:

```bash
#!/usr/bin/env bash

set -euo pipefail

cd /path/of/my/notes
(git add -A && git commit -m "$(date)" && git push) || echo "no changes"
```

For the curious, I'm using macOS's [`launchd`](https://en.wikipedia.org/wiki/Launchd) to run this script every fifteen minutes. After pushing to GitHub, it will trigger a [webhook](https://developer.github.com/webhooks/) that makes a request to my website API saying "there's new commits available, you should pull them" and that's what I do.

As soon as the server receives the hook request, it runs `git pull` on the server copy of the notes repository. Then, it runs a script that looks like this to convert the Notable notes to Hugo posts:


```javascript
const fs = require('fs-extra')
const { join } = require('path')
const yaml = require('js-yaml')
const slugify = require('slugify')

const dst = "/path/to/website/content/kb"
const src = "/path/to/original/notes"

await fs.remove(dst)
await fs.ensureDir(dst)

await fs.outputFile(
  join(dst, '_index.md'),
  `---
title: Knowledge Base
emoji: 🧠
---`
)

const files = await fs.readdir(src)

for (const index of files) {
  const path = join(src, index)
  if (fs.statSync(path).isDirectory()) {
    continue
  }

  const file = (await fs.readFile(path)).toString()
  let [frontmatter, content] = file.split('\n---')
  const meta = yaml.safeLoad(frontmatter)

  // Ignore notes with 'private' tag and deleted ones.
  if ((meta.tags && meta.tags.includes('private')) || meta.deleted) {
    continue
  }

  meta.date = meta.modified
  meta.publishDate = meta.created

  delete meta.modified
  delete meta.created
  delete meta.pinned

  content = content.trim()

  // Remove the initial heading.
  if (content[0] === '#') {
    content = content.substring(content.indexOf('\n')).trim()
  }

  // Check if there's some LaTeX going on so I know whether to require
  // Katex or not. You may not need this.
  if (content.match(/(\$\$.*?\$\$|\$.*?\$)/g)) {
    meta.math = true
  }

  // Check if there's some mermaid diagrams going on so I know whether to require
  // Mermaid or not. You may not need this either.
  if (content.includes('```mermaid')) {
    meta.mermaid = true
  }

  // Replace wiki links by true links that work with Hugo's.
  content = content.replace(/\[\[(.*?)\]\]/g, (match, val) => `[${val}](/kb/${slugify(val.toLowerCase())})`)

  // Outputs the final file!
  await fs.outputFile(
    join(dst, `${slugify(meta.title.toLowerCase())}.md`),
    `---\n${yaml.safeDump(meta, { sortKeys: true })}---\n\n${content.trim()}`
  )
}
```

After running this quick script, the website gets regenerated through hugo's commands. That's it. It's quite easy and the best part: I don't need to do anything manually. It usually goes smoothly. Nevertheless, it is good to check the logs from time to time.

Next steps? You can see them on my ~~knowledge base~~!