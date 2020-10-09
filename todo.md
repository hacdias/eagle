## GOAL: provide each post in MF2, JF2 and AS2

- **Current implementation**
  - Hugo generates AS2
    - Missing attachments for posts that are mainly images or videos
  - MF2 is generated on the fly using a library from the HTML
  - Webfinger is provided statically

- **Idea**
  - Create a tool (or provide on the fly with caching) that:
    - Generates .mf2 (single, lists)
    - Adapt to .jf2 (single)
    - Adapt to .as2 (single)

## GOAL: make the website fast

- Should I keep the current implementation (Hugo + a bunch of APIs)?
- Should I move to a completly server-side rendering? Write post!

**TODO**
- Setup test.hacdias.com
  - With @hacdias_test
  - With a testing repository too
  - On Hetzner
  - Check ActivityPub
  - Check webmention suite
  - Check with new config.yml WITH EVERYTHING FOR testing