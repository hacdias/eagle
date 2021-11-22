# Eagle

> Is there something that you think that could be pulled over to an external module?
> Please [let me know](https://github.com/hacdias/eagle/issues/new)!

This powers my website. It is open-source. However, I won't be supporting other people's use
cases as this is just a personal project for personal use. If you're interested in doing
something similar, I encourage you to take a look at the code.

This repository replaces the old, JavaScript based, [API](https://github.com/hacdias/eagle-js).

## Features

This is a non-extensive list of features.

- Micropub endpoint
- IndieAuth authorization and token endpoints
- Login via IndieAuth

### Visibility and Audience

The properties [visibility](https://indieweb.org/Micropub-extensions#Visibility) and [audience](https://indieweb.org/Micropub-extensions#Audience) are supported. The behaviour is as follows:

- `visibility=unlisted`: posts can be viewed, but they are not listed anywhere, except for `/unlisted`. `/unlisted` is only accessible to the administrator.
- `visibility=public`: anyone can access and is listed everywhere.
- `visibility=private`: only accessible to logged in users with the following constraints:
  - No `audience` set: all logged in users can view and is listed.
  - Specific `audience`: only the specified subset of users can view, listed under `/private`.

## License

MIT © Henrique Dias
