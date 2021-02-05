---
publishDate: "2016-11-24T00:00:00.000Z"
title: How to remove the query string from static assets on WordPress
---

When we were creating our theme and setting up our WordPress installation, we noticed that every single static asset had a query string in the end of the URL like `?ver=3.5` and we didn't want that because we were using Cloudflare and we were having problems updating the cache. So, we decided to remove that from our URLs. But how?

<!--more-->

There is a file on every WordPress theme that is very important to make internal changes to the theme. This file also allows us to modify the way WordPress works using only PHP. It is `functions.php` file. And to remove the query strings from static assets, we are going to use it.

Just open your `functions.php` file and copy and paste the following code:

```php
function remove_query_string($src)
{
    if (strpos($src, '?ver=')) {
        $src = remove_query_arg('ver', $src);
    }
    return $src;
}

add_filter('style_loader_src', 'remove_query_string', 10, 2);
add_filter('script_loader_src', 'remove_query_string', 10, 2);
```

In this code, we add two filters, pointing to the same function. This filter is attached to two hooks (`style_loader_src` and `script_loader_src`) which are called when the source path of the file is called.

The function `remove_query_string` is easy to understand: it takes one parameter (which is the initial source URL) and then, using the function strpos, we check if the URL has a query string. If it does, it is removed using the `remove_query_arg` function and then, we return our final URL, without the query string.