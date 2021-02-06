var AudioContext = window.AudioContext || window.webkitAudioContext

function createCanvas () {
  let canvas = document.createElement('canvas')
  canvas.width = window.innerWidth / 2
  canvas.height = window.innerHeight / 2

  document.body.appendChild(canvas)
  return canvas
}

function drawNoise (canvas) {
  let ctx = canvas.getContext('2d', { alpha: false })
  let idata = ctx.createImageData(canvas.width, canvas.height)
  let pix = idata.data
  let time = 0

  const generator = () => {
    for (let i = 0, n = pix.length; i < n; i += 4) {
      let c = 7 + Math.sin(i / 50000 + time / 7)
      pix[i] = pix[i + 1] = pix[i + 2] = 40 * Math.random() * c
      pix[i + 3] = 255
    }

    ctx.putImageData(idata, 0, 0)
    time = (time + 1) % canvas.height
  }

  setInterval(generator, 50)
}

function makeNoise () {
  let audioContext = new AudioContext()
  let bufferSize = 2 * audioContext.sampleRate
  let noiseBuffer = audioContext.createBuffer(1, bufferSize, audioContext.sampleRate)
  let output = noiseBuffer.getChannelData(0)

  for (let i = 0; i < bufferSize; i++) {
    output[i] = Math.random() * 2 - 1
  }

  let whiteNoise = audioContext.createBufferSource()
  whiteNoise.buffer = noiseBuffer
  whiteNoise.loop = true
  whiteNoise.start(0)
  whiteNoise.connect(audioContext.destination)
}

function start (event) {
  event.currentTarget.remove()
  drawNoise(createCanvas())
  makeNoise()
}
