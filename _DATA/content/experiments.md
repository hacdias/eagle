---
publishDate: "2020-05-30"
title: Experiments
updateDate: "2020-05-30"
---

<style>
body {
  background: #000 url(https://cdn.hacdias.com/uploads/fire.gif) bottom repeat-x;
  background-size: 100px;
}
</style>

<script>
function emojiToDataURL (emoji, size = '64') {
  const canvas = document.createElement('canvas')
  canvas.height = size
  canvas.width = size

  const ctx = canvas.getContext('2d')
  ctx.font = `${size}px serif`
  ctx.fillText(emoji, 0, size)
  return canvas.toDataURL()
}

document.body.style.cursor = `url(${window.emojiToDataURL('✨', 24)}), auto`
</script>

A galaxy, a black hole, a trash bin, whatever you wanna call it. This is a place for some weird
and creepy experiments. Weird things can happen, you can feel dizzy, be aware of dragons. 🐉
Be aware! Or they will bite you.

- 🔵 [Blue Screen of Death](/minisites/bsod/)
- 📡 [Glitch](/minisites/glitch/)
- 🗺 [Procedural Map Generator](/minisites/mapgen/)
- 🏳️‍🌈 [PixelColorMania](/minisites/pixelcolormania/)
- ⛈ [Thunderstorm](/minisites/thunderstorm/)
- 📺 [TV Noise](/minisites/tv-noise/)