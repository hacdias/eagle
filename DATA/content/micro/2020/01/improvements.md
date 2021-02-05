---
publishDate: "2020-01-25T23:45:22.486Z"
tags:
- meta
- design
---

Just made a few updates to my website:

- Added a [more](/more/) page, inspired by [@jlelse's](https://jlelse.blog/more/).
- Updated my [now](/now/) page to actually include what I'm doing right now.
- Updated the highlight theme to `swapoff`. It's a really pretty syntax highlight theme provided by [Chroma](https://swapoff.org/chroma/playground/), the library Hugo is using. The best thing is: using the `invert()` CSS filter it keeps looking good. This way, I have good syntax highlight on both the light and dark themes.
- Added the category of the post right besides the publish date.
- Now I'm only showing notes, articles and replies on the homepage.
- I have a [all](/all) page dedicated to show every post category.

I just noticed the website on the dark mode flickers sometimes (from the light to the dark theme). I know this is caused by the fact that I'm using JavaScript to pick the theme depending on the OS choice + the manual overriding done by the user using the option on the bottom of the page.

One option to remove this problem would be to just follow the user's OS choice (either dark or light) and there wouldn't be a manual override. Unfortunately, this wouldn't let the users pick a different theme if they prefer to read websites on a different light - no pun intended.