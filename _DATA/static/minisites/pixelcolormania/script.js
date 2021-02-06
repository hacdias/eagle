const dpi = window.devicePixelRatio || 1

const colors = [
  '#E70000',
  '#FF8C00',
  '#FFEF00',
  '#00811F',
  '#0044FF',
  '#760089'
]

let options = {
  size: 100,
  lines: false,
  overlay: false
}

let canvas = createCanvas()
let interval = null

function createCanvas () {
  let canvas = document.createElement('canvas')
  canvas.width = window.innerWidth * dpi
  canvas.height = window.innerHeight * dpi

  document.body.appendChild(canvas)
  return canvas
}

function draw (canvas, { size = 10, overlay = false, lines = false }) {
  let ctx = canvas.getContext('2d')
  let factor = overlay ? 1 : size
  let w = canvas.width / factor
  let h = canvas.height / factor

  const gen = () => {
    let m = overlay ? 1 : size
    let x = lines ? 0 : (Math.floor(Math.random() * w) * m)
    let y = Math.floor(Math.random() * h) * m
    return { x, y }
  }

  const generator = () => {
    const { x, y } = gen()
    ctx.fillStyle = colors[Math.floor(Math.random() * colors.length)]
    ctx.fillRect(x, y, lines ? canvas.width : size, size)
  }

  return setInterval(generator, 5)
}

interval = draw(canvas, options)

function reset () {
  document.body.removeChild(canvas)
  canvas = createCanvas()
  redo()
}

function toggle (event) {
  if (interval) {
    clearInterval(interval)
    interval = null
    event.currentTarget.innerHTML = 'Resume'
  } else {
    interval = draw(canvas, options)
    event.currentTarget.innerHTML = 'Stop'
  }
}

function redo () {
  clearInterval(interval)
  interval = draw(canvas, options)
}

function set (event, what) {
  options[what] = event.currentTarget.checked
  if (interval) redo()
}

function setSize (event) {
  options.size = parseInt(event.currentTarget.value)
  if (interval) redo()
}
