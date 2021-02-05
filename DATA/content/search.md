---
emoji: "\U0001F50E"
noMentions: true
title: Search
---

<style>
  .result div {
    display: grid;
    grid-template-columns: auto 100px;
    grid-gap: 1rem;
  }

  .result time {
    font-variant-numeric: tabular-nums;
    letter-spacing: -.012em;
    white-space: pre;
    text-align: right;
  }

  .result span {
    font-weight: bold;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result {
    margin: 1rem 0;
  }

  .result p {
    margin-top: 0.25rem;
    font-size: var(--f6);
  }
</style>

This search is powered by [MeiliSearch](https://docs.meilisearch.com/), an open-source, and super fast search-engine
that I am self-hosting alongside with the rest of the API of this website.

<form id="search-form" class="inline-form">
  <input aria-label="Search terms" required type="text" name="query" placeholder="What do you want to search for?"></input>
  <input type="submit" value="Search" style="background-color: var(--action); border-color: var(--action)"></input>
</form>

<div id="search-results"></div>

<p id="more-results" class="dn" style="display:flex; font-size:var(--f6)">
  <button style="margin-left: auto" onclick="search(true)">More results…</button>
</p>

<script>
  const input = document.querySelector('#search-form input[type="text"]')
  const results = document.querySelector('#search-results')
  const more = document.querySelector('#more-results')
  let query
  let page = 0

  function fromHex (h) {
    var s = ''
    for (var i = 0; i < h.length; i+=2) {
      s += String.fromCharCode(parseInt(h.substr(i, 2), 16))
    }
    return decodeURIComponent(escape(s))
  }

  async function search (nextPage) {
    if (nextPage) {
      page++
    }

    try {
      var search = new URLSearchParams(window.location.search);
      search.set("q", query);
      search.set("p", page);
      const url = `../search.json?${search.toString()}`
      const res = await fetch(url)
      const data = await res.json()
      more.classList.add('dn')

      if (!Array.isArray(data)) {
        throw new Error('response is not array')
      }

      if (data.length === 0) {
        results.innerHTML = '<p>No results found for this search…</p>'
        return
      }

      if (page === 0) {
        results.innerHTML = ''
      }

      for (let { _formatted: { content, tags, id, title, date} } of data) {
        id = '..' + fromHex(id)
        title = title || 'A note'
        if (date) {
          date = new Date(date)
          const year = date.getFullYear()
          const month = (date.getMonth()+1).toString().padStart(2, '0')
          const day = date.getDate().toString().padStart(2, '0')
          date = `${year}-${month}-${day}`
        } else {
          date = 'Unknown date'
        }
        results.innerHTML += `<div class="result">
    <a href="${id}" class="no-link">
      <div>
        <span>${title}</span>
        <time>${date}</time>
      </div>
      <p>…${content}…</p>
    </a>
  </div>`
      }

      if (data.length == 20) {
        more.classList.remove('dn')
      }
    } catch (err) {
      results.innerHTML = `<p style="color:red">An error occurred: ${err.toString()}</p>`
    }
  }

  document.getElementById('search-form').addEventListener('submit', event => {
    event.preventDefault()
    page = 0
    query = input.value
    search()
  })
</script>

<noscript>Unfortunately, this page doesn't work without JavaScript enabled 😭</noscript>