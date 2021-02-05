---
publishDate: "2020-03-06T11:57:08.634Z"
---

That's a great way to put it. Thanks for sharing how you're doing it. I'm using the HTML [`picture`](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/picture) element to serve pictures in different formats and sizes. Right now, besides the original, Hugo's generating two more that are compressed and resized.

With the new Photos page, I want to create a gallery view so I would also need to have a thumbnail... And this complexity of generating all this different kind of pictures is what's stopping me now to do it the way you're doing. I'd certainly also want to have WebP versions of them. I just don't because Hugo does not support converting to WebP yet.

This is definitely not impossible, it's doable. But is it the best option? What if I want to suddenly change the sizes in the gallery? I'll need to regenerate everything... Maybe thinking too far ahead.