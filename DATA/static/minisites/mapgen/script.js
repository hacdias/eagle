const Noise = window.noise
const frame = document.getElementById('container')

function createCanvas (dpi = 1) {
  let canvas = document.createElement('canvas')

  const width = parseInt(window.getComputedStyle(frame).width)
  const height = parseInt(window.getComputedStyle(frame).height)

  canvas.width = width
  canvas.height = height
  canvas.style.width = width
  canvas.style.height = height

  frame.innerHTML = ''
  frame.appendChild(canvas)
  return canvas
}

function lerp (v0, v1, t) {
  return (t - v0) / (v1 - v0)
}

function biome (e) {
  if (e < 0.3) {
    return '#2c52a0'
  } else if (e < 0.4) {
    return '#3766c8'
  } else if (e < 0.45) {
    return '#d0d080'
  } else if (e < 0.55) {
    return '#589619'
  } else if (e < 0.60) {
    return '#426220'
  } else if (e < 0.70) {
    return '#5c453e'
  } else if (e < 0.90) {
    return '#4d3b39'
  } else {
    return '#ffffff'
  }
}

function build ({ scale = 250, octaves = 5, persitance = 0.5, lacunarity = 2.5 }) {
  console.time('BUILD')
  const dpi = window.devicePixelRatio || 1
  const canvas = createCanvas(dpi)
  const ctx = canvas.getContext('2d', { alpha: false })

  Noise.seed(Math.random())

  console.time('GEN_ELEV')

  const height = Math.floor(canvas.height / dpi)
  const width = Math.floor(canvas.width / dpi)

  let elev = [...Array(height)].map(e => Array(width))
  console.timeEnd('GEN_ELEV')

  let min = Number.POSITIVE_INFINITY
  let max = Number.NEGATIVE_INFINITY

  console.time('NOISE')

  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      let amplitude = 1
      let frequency = 1
      let noise = 0

      for (let i = 0; i < octaves; i++) {
        let sX = x / scale * frequency
        let sY = y / scale * frequency
        noise += Noise.simplex2(sX, sY) * amplitude

        amplitude *= persitance
        frequency *= lacunarity
      }

      max = Math.max(noise, max)
      min = Math.min(noise, min)
      elev[y][x] = noise
    }
  }

  console.timeEnd('NOISE')

  console.time('NOISE2')

  for (let [y] of elev.entries()) {
    for (let [x] of elev[y].entries()) {
      let n = lerp(min, max, elev[y][x])

      ctx.fillStyle = biome(n)
      ctx.fillRect(x * dpi, y * dpi, dpi, dpi)
    }
  }
  console.timeEnd('NOISE2')

  console.timeEnd('BUILD')
}

build({})
