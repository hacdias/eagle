---
description: |
  After updating the structure of my URLs, I now needed to start enabling some IndieWeb features... here's how to enable IndieAuth!
publishDate: "2019-12-30T23:50:00.000Z"
syndication:
- https://twitter.com/hacdias/status/1211797283525812224
tags:
- indieweb
- indieauth
title: Enabling IndieAuth on my website
---

This is going to be a quick post — I hope. 

After updating the [structure of the URLs](/articles/2019/12/url-structure/) on my website, I felt like the next step would be to implement [IndieAuth](https://indieauth.net/) because almost everything on the IndieWeb needs that.

<!--more-->

But first, what is IndieAuth? According to [IndieWeb](https://indieweb.org/IndieAuth):

>  IndieAuth  is a federated login protocol for  Web sign-in , enabling users to use their own  domain  to sign in to other sites and services. IndieAuth can be used to implement **OAuth2 login** AKA **OAuth-based login**.  

So basically you can sign in with our own domain, being your domain your username and password and your identity online, which is… kind of awesome if we think about it since a domain is unique and you can make it yours.

### How does it work?

Really quickly, let’s go through how it works:

1. The user — you! — fills theirs URL in the login form
2. The app fetches the URL, looks for the authorisation endpoint and redirects the user to their authorisation endpoint
3. The user authenticates and the endpoint generates a temporary authorisation token the app can use to verify the user’s identity

It’s quite an interesting roundtrip but that’s what most applications nowadays do anyways…

### How did you enable it?

So… I didn’t want to be messing with IndieAuth servers and I didn’t want to setup my own so I’m relying on [IndieAuth.com](https://indieauth.com/) for now. Their service allows you to login with some supported services that you provide through `rel=me` links in your home page.

For example, if I have:

```html
<a href=“https://github.com/hacdias” rel=“me”>GH</a>
```

Somewhere in the body of my page, or:

```html
<link rel=“me” href=“https://github.com/hacdias”>
```

In the `<head>`,  then the IndieAuth.com service will be able to connect to GitHub and I’ll be able to login through GitHub. This is all it takes! Yes, that’s true. Simple as it is.

**Remember** that IndieAuth.com is just a service that provides IndieAuth for you. You can setup your own server or use [other methods](https://indieweb.org/IndieAuth#IndieWeb_Examples). I’ll probably setup my own method soon so it’s easier.

### Discovery, discovery… where are you?

So there’s one step missing, besides adding the `rel=“me”` links to your page. You need to make your authorisation endpoint discoverable! But how do you do that?

You just need to add some more `<link>` s to the head of your HTML file! For using IndieAuth.com, these are the endpoints:

```html
<link rel=“authorization_endpoint” href=“https://indieauth.com/auth”>
<link rel=“token_endpoint” href=“https://tokens.indieauth.com/token”>
``` 

Et voilà!