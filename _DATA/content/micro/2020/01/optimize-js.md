---
publishDate: "2020-01-26T10:38:10.206Z"
tags:
- design
- javascript
- optimization
---

Just fixed the flickering issue by inlining the required JavaScript, which is a really small snippet. If you're curious about the code I'm using to toggle between the dark and light themes, following the user's choice and falling back to the OS settings, here it is:

```javascript
const mql = window.matchMedia('(prefers-color-scheme: dark)');

function toggleT (to) {
  localStorage.setItem('t', to);
  theme(mql);
}

function theme (query) {
  const userOption = localStorage.getItem('t');

  document.body.id = userOption === null
    ? (query.matches ? 'dark' : '')
    : (userOption === 'd' ? 'dark' : '');
}

mql.addListener(theme);
theme(mql);
```