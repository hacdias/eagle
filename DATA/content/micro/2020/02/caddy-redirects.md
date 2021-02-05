---
publishDate: "2020-02-16T14:32:39.871Z"
tags:
- meta
- hugo
- development
---

After publishing the post to which I'm replying to, [@jlelse](https://jlelse.blog/) contacted me and I noticed the [`import`](https://caddyserver.com/v1/docs/import) directive in Caddy can be used to import files:

> import allows you to use configuration from another file or a reusable snippet. It gets replaced with the contents of that file or snippet.

So I just decided to build the redirects file using Hugo itself. First of all, I needed to import a lot of redirects as aliases because I had them in a separate file, but this way it's much better. After that, I needed to add a new output type to Hugo's config:

```yaml
disableAliases: true

outputFormats:
  redir:
    mediaType: text/plain
    baseName: redirects
    isPlainText: true
    notAlternative: true

outputs:
  home:
    - redir
```

Then, I created a `layouts/index.redir.txt` file with the following content:

```html
{{- range $p := .Site.Pages -}}
{{ range .Aliases }}
{{  . | printf "%-70s" }}	{{ $p.RelPermalink -}}
{{ end -}}
{{- end -}}
```

This is mostly what you can see on this [commit](https://github.com/gohugoio/hugoDocs/commit/c1ab9894e8292e0a74c43bbca2263b1fb3840f9e) of the official hugo docs for their netlify redirects. With this, my Hugo website does not build any HTML aliases (`disableAliases`), but creates a file on the root called `redirects.txt` which you see [here](https://hacdias.com/redirects.txt). I can just block the access through Caddy but there's no reason I should do so. Is there?

On Caddyland, I just added this snipped:

```txt
hacdias.com {
  root /the/public/path/

  redir 301 {
    import /the/public/path/redirects.txt
  }
}
```

And voilá! It works! But now you ask: what if we change the redirects file and we don't wanna have any downtime? Just configure your Micropub entrypoint or whatever software you're using on the backend to do a config hot reload by executing the following command:

```txt
pkill -USR1 caddy
```

There it is! 301 redirects working flawlessly!