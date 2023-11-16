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

## License

MIT © Henrique Dias
