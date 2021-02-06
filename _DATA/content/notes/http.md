---
title: HTTP
---

The **Hypertext Transfer Protocol** (HTTP) is an [Application Layer](/notes/application-layer/) protocol.

## Verbs

- `GET` - request a document.
- `POST` - update a document.
- `PUT` - store a document.
- `DELETE` - delete a document.
- `HEAD` - request a document's header.

Here's a GET example:

```
GET /somedir/page.html HTTP/1.1
Host: tecnico.ulisboa.pt
Connection: close
User-agent: Mozilla/4.0
Accept-language: en
```

And a sample response:

```
HTTP/1.1 200 OK
Connection: close
Date: Thu, 03 Jul 2013 12:00:15 GMT
Server: Caddy
Last-Modified: Sun, 5 May 2013 09:23:24 GMT
Content-Length: 12234
Content-Type: text/html

<html>
  ...
```

## HTTP 1.0

- One [TCP](/notes/tcp/) connection for each request.

## HTTP 1.1

- One [TCP](/notes/tcp/) connection can make one or more requests.
- Requests can have arbitrary size.
- Blockage is still possible (*Head of Line*): replies arrive on the same order.

## HTTP/2

- Uses binary format instead of text. The headers and request verbs are the same, but they're encoded differently in a more efficient manner.
- Bidirectional data flux.
- Streams!