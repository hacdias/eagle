---
description: |
  Caddy is a lightweight, fast, general-purpose, cross-platform HTTP/2 web server with automatic HTTPS with support for plugins. I've created some.
publishDate: "2015-09-12T12:10:00.000Z"
tags:
- projects
title: Caddy Plugins
updateDate: "2021-01-28T12:10:00.000Z"
---

[Caddy](https://caddyserver.com) is a HTTP/2 web server with automatic HTTPS. It's easy to use, has no dependencies and it is already production-ready. One of the biggest advantages of Caddy is that it is relatively simple to create plugins that extend its own native functionalities.

<!--more-->

Despite having contributed few times to Caddy itself, I've managed to create some useful plugins for the first version of Caddy. In the meanwhile, a second version of Caddy was released and I chose not to update my plugins, discontinuing them. Some of the reasons regard the usage, popularity, "easyness" of migration and alternatives.

+ **http.filebrowser** is an implementation of my [File Browser project](/articles/2016/06/filebrowser/) as a plugin for Caddy.
+ **http.hugo** is a web interface for [Hugo](https://gohugo.io) static website generator.
+ **http.jekyll** is a web interface for [Jekyll](https://jekyllrb.com/) static website generator.
+ [**http.minify**](https://github.com/hacdias/caddy-v1-minify) implements minification on-the-fly for CSS, HTML, JSON, SVG and XML. It uses [tdewolff's minify library](https://github.com/tdewolff/minify).
+ [**http.webdav**](https://github.com/hacdias/caddy-v1-webdav) implements WebDAV capabilities with support for path restriction rules and users.
+ [**hook.service**](https://github.com/hacdias/caddy-v1-service) implements of [github.com/kardianos/service](https://github.com/kardianos/service) to create services. Supports Windows XP+, Linux/(systemd | Upstart | SysV), and OSX/Launchd.