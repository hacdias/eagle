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

## Configuration and Assumptions

Eagle makes a certain amount of assumptions regarding your Hugo website. That is the only way it will work properly. The following sections try to document all assumptions and required configuration. It can also be useful to look at my own website's [source code](https://github.com/hacdias/hacdias.com).

### Hugo Posts

Hugo posts are assumed to be bundles and not plain `.md` files. For example, `about.md` is invalid, you must have `about/index.md`.

### Hugo Configuration

Eagle takes some of the configuration directly from your Hugo's website. It supports a single file configuration in any of the formats supported by Hugo (JSON, TOML, YAML). The following parts are used:

```toml
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
```

It does not support `config` directory, or multi language configuration.

### Hugo Sections

Eagle assumes a very important `posts` section. This section contains all of your main, dated, posts, such as articles. Inside the `posts` directory, there are directories per year. Inside each year directory, there is a directory per post.

The post on `/posts/2023/02/10/my-post/index.md` with `2023-02-10` in the `date` frontmatter field is assumed to be published at `/2023/02/10/my-post/`.

Micropub posts are created following this assumption.

### Hugo Taxonomies

Eagle assumes that taxonomy pages are the only page lists. The `categories` taxonomy is handled differently. The category page at `/categories/articles/_index.md` is assumed to be published at `/articles/`.

### HTML Files

Eagle expects a `entry-id` meta element in your Hugo's website HTML output. This will be used to do inverse mapping from Permalink to the Entry ID. This may be changed in the future by reverse engineering the assumed permalinks.

```html
{{ with .File }}
  <meta name='entry-id' content='{{ .Dir }}'>
{{ end }}
```

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
