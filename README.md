# Eagle

> Is there something that you think that could be pulled over to an external module?
> Please [let me know](https://github.com/hacdias/eagle/issues/new)!

This powers my website. It is open-source. However, I won't be supporting other people's use
cases as this is just a personal project for personal use. If you're interested in doing
something similar, I encourage you to take a look at the code.

## Install

```console
go install go.hacdias.com/eagle@latest
```

Or:

```console
docker pull ghcr.io/hacdias/eagle:latest
```

## Configuration

### Hugo Configuration

Eagle takes some of the configuration directly from your Hugo's website. It supports a single
file configuration in any of the formats supported by Hugo (JSON, TOML, YAML). The following
parts are used:

```toml
# Used by Hugo and by Eagle for search results.
paginate = 15

[params]
  [params.author]
    # Optional user's information for IndieAuth.
    name = 'John Smith'
    email = 'john@smith.com'
    photo = 'https://smith.com/avatar.png'
    # Optional user's handle for WebFinger. Disabled if empty.
    handle = 'johnsmith'
```

### Templates

Hugo website must produce a `/eagle/index.html` page. This page will be used as the template
for the interface. The page should contain a `<eagle-page>` element, which Eagle will replace
by the correct content:

```html
<eagle-page>
  <p>⚠️ This page is only functional when in use with <a href='https://github.com/hacdias/eagle' rel='noopener noreferrer'>Eagle</a>.</p>
</eagle-page>
```

Then, the Hugo website must have a `eagle` directory containing the following templates:

- `admin-bar.html`: admin bar injected at the top of every page when user is logged in.
- `authorization.html`: authorization page for IndieAuth.
- `error.html`: error page.
- `login.html`: login page.
- `panel.html`: panel page.
- `panel-guestbook.html`: guestbook comments moderation page.
- `panel-tokens.html`: IndieAuth tokens management page.
- `search.html`: search page.

At the moment, it is best to check the source code to see what variables are available
in each template. I may add an example at some point, or a link to when I re-open source
my website's code.

## License

MIT © Henrique Dias
