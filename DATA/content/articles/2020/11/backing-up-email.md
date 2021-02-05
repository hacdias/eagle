---
publishDate: "2020-11-02T09:00:00.000+01:00"
tags:
- ownyourdata
- mondayletter
- backup
title: How to effectively backup your emails
---

For quite some time, I have been setting up systems to backup my data of my computer, as well as fetching data from services, such as Trakt, Last.fm or GoodReads. There's always one kind of service that has been on the back of my mind for a while to backup, but I've never got the time, nor the will to do so: email!

Email is fundamental nowadays and it is the basis of Internet communication. Almost all online services require an email, and even though we use it virtually every day for the most varied services and uses, it is not the easiest thing to backup.

Fortunately, I recently came across [this post](https://www.artemix.org/blog/backing-up-e-mails-from-an-imap-server) by Diane where they explain how they are backing up their emails using `getemail`. 

First of all, start by installing `getmail`. In my case, since I use macOS, I will just use the pre-built brew package. For other platforms (since it uses Python), I recommend taking a look at their [website](http://pyropus.ca/software/getmail/).

```
brew install getmail
```

Now that the tool is installed, it's time to configure it. First, create the directory where we're going to store the configuration. By default, that directory is `~/.getmail` and the default configuration file is `getmailrc`. 

```
mkdir ~/.getmail
```

For my specific case I am creating different configuration files and I want to archive my Gmail data at `~/gmail-archive`. To do so, I create that directory, as well as three other directories inside of it:

```
mkdir ~/gmail-archive
mkdir {cur,tmp,new}
```

This will ensure our directories are compatible with the [`maildir`](https://cr.yp.to/proto/maildir.html) format we're going to use to store the emails. Don't worry! They're all plaintext and easily searchable from the console line or any other tool you might want!

Now, let's create a configuration file for Gmail at `~/.getmail/gmail.conf`. On it, just copy the following configuration and change the user, password, and path to match what you want.

```toml
[retriever]
type = SimpleIMAPSSLRetriever
server = imap.gmail.com
username = <your-user>@gmail.com
password = <your-password>
mailboxes = ("[Gmail]/All Mail",) # To pull all emails
port = 993

[destination]
type = Maildir
path = ~/gmail-archive

[options]
read_all = false # Do not mark emails as read
```

As you can see, I'm using IMAP retriever. However, there's a few [other options](http://pyropus.ca/software/getmail/configuration.html#conf-retriever), with and without (don't!) SSL.

If you're using a service other than GMail, please change the `mailboxes` variable too. You can write `mailboxes = ALL` and it will download all the available mailboxes. By default, it downloads `INBOX` only.

Now, to fetch your email, just run:

```
getmail -r ~/.getmail/gmail.conf
```

*Et voilá!* You have your email backed up! You now can create more configuration files if you want for different accounts and set up a cronjob to execute this command.

I really hope this will be as useful for you as it was for me!