# Eagle

> Is there something that you think that could be pulled over to an external module Please [let me know](https://github.com/hacdias/eagle/issues/new)!

This powers my website. It is open-source. However, I won't be supporting other people's use cases as this is just a personal project for personal use. If you're interested in doing something similar, I encourage you to take a look at the code.

## Install

```console
go install go.hacdias.com/eagle@latest
```

Or:

```console
docker pull ghcr.io/hacdias/eagle:latest
```

## Features

- Single user system.
- Receive and send [Webmentions](https://webmention.net/). Incoming must be configured via [Webmention.io](https://webmention.io).
- Comment endpoint, allowing to directly submit comments via the website.
- [IndieAuth](https://indieauth.spec.indieweb.org/) OAuth server to login elsewhere with your website.
- Notifications (e.g. from Webmentions) via custom Telegram bot.
- Media storage on [Bunny CDN](https://bunny.net).
- Media resizing and compression via [ImgProxy](https://imgproxy.net/).
- Serve the website as a TOR onion service.
- [MeiliSearch](https://www.meilisearch.com/) integration for website search.
- [POSSE](https://indieweb.org/POSSE) to Mastodon, Bluesky and IndieNews.
- AT Protocol integrations with [arabica.social](https://arabica.social), [Standard.site](https://standard.site) and Bluesky.
- Reverse location information for post metadata.
- Miniflux blogroll integration.
- WebArchive integration to archive links present in the content of new posts.
- Implemented integrations but no longer used: [Linkding](https://linkding.link/), [XRay](https://xray.p3k.io/).

## Configuration and Assumptions

Eagle makes a certain amount of assumptions regarding your Hugo website. That is the only way it will work properly. The following sections try to document all assumptions and required configuration. It can also be useful to look at my own website's [source code](https://github.com/hacdias/hacdias.com).

### General Assumptions

- The website is served at the domain's root.
- [Page bundles](https://gohugo.io/content-management/page-bundles/) are used for all pages. The source code of `/about` is at `/content/about/index.md`.
- The following two taxonomies exist:
  - `tags` which are used for search indexing. Pages are published at `/tags/{tag}/`.
  - `categories`, which are post categories. Pages are published at `/{category}/`.
- The `posts` section is a special section. It contains all main, dated, posts, such as articles. Inside the `posts` directory, there are directories per year. Inside each year directory, there is a directory per post. The post on `/posts/2023/02/10/my-post/index.md` with `2023-02-10` in the `date` frontmatter field is assumed to be published at `/2023/02/10/my-post/`.

### Hugo Configuration

Eagle uses some of the configuration directly from your Hugo's website in order to prevent duplication. It supports a single file configuration in any of the formats supported by Hugo (JSON, TOML, YAML). The following parts are used:

```toml
# The domain at which the website is served.
baseURL = 'https://example.com/'

# The title of the website.
title = 'My Website'

# Used by the location plugin to fetch the location in this language.
locale = 'en'

# Used by Eagle for the search results, in order to match
# the same used for Hugo when listing posts.
[pagination]
  pagerSize = 15

# Used by Eagle to determine if a certain page is a page list.
# Only taxonomies are considered lists for Eagle.
[taxonomies]
  tag = 'tags'
  category = 'categories'

[params]
  [params.author]
    # Optional user's information for IndieAuth.
    name = 'John Smith'
    email = 'john@smith.com'
    photo = 'https://smith.com/avatar.png'
    # Optional user's handle for WebFinger. Disabled if empty.
    handle = 'johnsmith'
  [params.site]
    # Used for the standard.site integration.
    description = 'Lorem Ipsum'
```

It does not support `config` directory, or multi language configuration.

### Templates

The following pages must be produced by your Hugo website:

- `404.html` for 404 and other errors.
- `/search/index.html` **if** search is enabled through Eagle.

These pages must contain a `<eagle-page>` element, which Eagle will replace by the correct content. For example:

```html
<eagle-page>
  <h1>404 Not Found</h1>
  <p>Page could not be found.</p>
</eagle-page>
```

Then, the Hugo website must have a `eagle` directory containing the following templates:

- `error.html` for error page, which will replace content in `404.html`.
- `search.html` for search page, which will replace content in `/search/index.html`.

At the moment, it is best to check the source code to see what variables are available in each template.

## License

MIT © Henrique Dias
