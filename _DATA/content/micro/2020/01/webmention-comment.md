---
publishDate: "2020-01-25T15:39:17.084Z"
tags:
- indieweb
- webmentions
- meta
---

I just added a little form in the webmentions section on the posts so you can send your webmention manually if it didn't hit my website. Or, you can also use the button 'Write a comment' to create a comment (that can be anonymous) through the [comment parade](https://commentpara.de/) service.

For the curious ones out there, the code is simple. Please remember that I'm using [Go templates](https://golang.org/pkg/text/template/) with [Hugo](https://gohugo.io/):

```html
<form action="https://webmention.io/hacdias.com/webmention" method="post">
  <input name="source" placeholder="Have you written a response? Paste its URL here!" type="url" required>
  <input name="target" value="{{ .Permalink }}" type="hidden">
  <input value="Send Webmention" type="submit">
</form>

<form method="get" action="https://quill.p3k.io/" target="_blank">
  <input type="hidden" name="dontask" value="1">
  <input type="hidden" name="me" value="https://commentpara.de/">
  <input type="hidden" name="reply" value="{{ .Permalink }}">
  <input type="submit" value="Write a comment">
</form>
```