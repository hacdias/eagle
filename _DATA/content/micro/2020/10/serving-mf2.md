---
publishDate: "2020-10-13T13:47:55Z"
replyTo: https://jan.boddez.net/notes/514f198e74
---

Yeah, that's true. But it was a tiny bit more complicated. I was serving the microformats JSON directly instead of the HTML and I forgot to add the domain to the parser I was using. Simply ended up removing that. There's no point (I think) in me serving jf2 or mf2 JSON directly. Besides HTML with Microformats, I'm also serving ActivityStreams and I think that, unfortunately, that's what X-Ray picks up, and not the HTML, so I'm trying to make sure it is as reliable as possible.