---
home: true
publishDate: "2020-05-14T21:36:29.879Z"
tags:
- automation
- dns
- tools
- dotfiles
---

Inspired by @jlelse's post on [DNSControl](https://jlelse.blog/links/2020/05/dnscontrol/), I moved my few DNS configurations to use that auto-deployable system and I must say I am amazed. I've known a few other systems for such kind of automation like Ansible, but never used them. In the meanwhile, I also started using GNU's [stow](https://www.gnu.org/software/stow/) to manage my dotfiles. It's much better than my previous system where I had a script that just _copied_ the dotfiles over. This way they're symlinked so the changes will be reflected with ease. I'm happy with it.