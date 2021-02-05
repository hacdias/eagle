---
publishDate: "2020-01-30T23:40:31.938Z"
tags:
- meta
- fixes
updateDate: "2020-01-30T23:45:37.147Z"
---

Just solved the deadlock! I'm currently using the [p-limit](https://www.npmjs.com/package/p-limit) package to limit the number of concurrent actions made to the website source. Basically, inside a function wrapped by that limitation, I was waiting for another function that would require the limit to be complete! Of course, that would create a never-ending deadlock. Fixed now!

On a second thought: I don't actually like the structure of the internal code I use to process all of this. Maybe I should rearrange some things to make them... better.