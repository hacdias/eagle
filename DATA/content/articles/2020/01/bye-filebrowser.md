---
description: After 5 years of maintaining this project and failing due to my limited time, it was time to say goodbye to File Browser.
publishDate: "2020-01-23T09:10:00.000Z"
syndication:
- https://twitter.com/hacdias/status/1220274794068959232
tags:
- opensource
- go
- lessons
title: Goodbye File Browser!
---

In 2015, I started a project called `http.hugo`, which was a just a simple plugin for [Caddy](https://caddyserver.com/), a really fast web server built with Go with automatic HTTPS.  At the time, the plugin was exclusive for Caddy and it provided a simple UI to edit your files in the server, rebuild the website and so on. They were just simple features.

<!--more-->

The plugin that initially only aimed to edit Hugo files started being used by other people as a file manager for generic static website generators. With that being said, I made a few updates and released three different plugins:  `http.filemanager`, `http.jekyll`  and `http.hugo`, being the last two a “remix” of the first one with static website generator support.

At this time, File Manager was not only just a program to edit, create, upload and delete files. It had tons of different features, such as:

* Run pre-determined commands on the server;
* Custom CSS;
* Metadata parsing for Hugo/Jekyll files;
* Different permissions per user;
* Hooks for commands on different types of actions;
* Etc.

In addition, due to the move from my GitHub user to an organisation, we had to rename the project File Browser because the org ‘File Manager’ was not available on GitHub. However, that wasn’t such a big problem.

The project evolved and I started university and I also started working at [Protocol Labs](/articles/2018/10/working-at-protocol-labs/), which has been a terrific opportunity. However, that’s not why I’m here today.

I saw I didn’t have much time for the project, so I opened an issue [looking for maintainers](https://github.com/filebrowser/filebrowser/issues/532). Even though a lot of people commented saying that they could maintain the project or fix some bugs, not even a 1/10 of them opened a PR. I waited a year and I decided to archive the repository. No one was maintaining it. I didn’t have time. There were bug reports everywhere.

After closing it, I’ve already received emails from some people thanking me for the project, saying that they’re sorry to see the project end. I even received an email from [@ jeromewu](https://github.com/jeromewu), where they offered to maintain the project. I added them as owners to the organisation. So far, no progresses.

I think one of the biggest mistakes I’ve done on this journey was to support superfluous features. Things that only one or two people wanted that caused a lot of bugs. Why do we need custom CSS? Why do we need command hooks? Why did we need to share files? This was meant to be **just** a lightweight File Manager for private use. Nothing else. Not a full featured alternative to [OwnCloud](https://owncloud.org/) for example.

Anyways, bye bye File Browser 👋 I hope that eventually someone actually takes care of the project if they really want to.

And a note to myself: rely on simplicity and do not add features just because a few people wanted. Ask myself first:

* Is it in the scope of the project?
* Is it important?
* What value would it bring?
* Do I have time to maintain one more feature?