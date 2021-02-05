const src = document.currentScript.src.split('/').slice(0, -1).join('/')

function random (min, max) {
  return Math.random() * (max - min + 1) + min
}

class Thunderstorm {
  constructor (elem, opts = {}) {
    opts = Object.assign({}, {
      rainLines: 500,
      speedRainLines: 25,
      rainDrops: 500
    }, opts)

    elem.innerHTML = ''

    this.rainDrops = this._createCanvas()
    this.rainLines = this._createCanvas()
    this.lightening = this._createCanvas()

    elem.appendChild(this.rainDrops.canvas)
    elem.appendChild(this.rainLines.canvas)
    elem.appendChild(this.lightening.canvas)

    this.opts = opts

    this._initRainLines()
    this._initRainDrops()
    this._initLightening()
    window.addEventListener('resize', this._initRainLines.bind(this))

    let audio = new window.Audio(`${src}/assets/rain.mp3`)
    audio.loop = true

    audio.play().catch(() => {
      document.addEventListener('click', () => { audio.play() })
    })
  }

  _createCanvas () {
    let canvas = document.createElement('canvas')
    canvas.width = window.innerWidth
    canvas.height = window.innerHeight

    window.addEventListener('resize', () => {
      canvas.width = window.innerWidth
      canvas.height = window.innerHeight
    })

    return {
      canvas: canvas,
      ctx: canvas.getContext('2d'),
      data: []
    }
  }

  _initRainLines () {
    let { canvas } = this.rainLines

    for (var i = 0; i < this.opts.rainLines; i++) {
      this.rainLines.data[i] = {
        x: random(0, canvas.width),
        y: random(0, canvas.height),
        length: Math.floor(random(1, 830)),
        opacity: Math.random() * 0.2,
        xs: random(-2, 2),
        ys: random(10, 20)
      }
    }
  }

  _clearRainLines () {
    let { canvas, ctx } = this.rainLines
    ctx.clearRect(0, 0, canvas.width, canvas.height)
  }

  _drawRainLines (i) {
    let { ctx, data } = this.rainLines

    ctx.beginPath()
    var grd = ctx.createLinearGradient(0, data[i].y, 0, data[i].y + data[i].length)
    grd.addColorStop(0, 'rgba(255,255,255,0)')
    grd.addColorStop(1, 'rgba(255,255,255,' + data[i].opacity + ')')

    ctx.fillStyle = grd
    ctx.fillRect(data[i].x, data[i].y, 1, data[i].length)
    ctx.fill()
  }

  _animateRainTrough () {
    this._clearRainLines()

    let { canvas, data } = this.rainLines

    for (let i = 0; i < this.opts.rainLines; i++) {
      if (data[i].y >= canvas.height) {
        data[i].y = canvas.height - data[i].y - data[i].length * 5
      } else {
        data[i].y += this.opts.speedRainLines
      }
      this._drawRainLines(i)
    }
  }

  _initRainDrops () {
    let { canvas } = this.rainLines

    this.rainDrops.data = [...Array(this.opts.rainDrops)].map(() => ({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      l: Math.random() * 1,
      xs: -4 + Math.random() * 4 + 2,
      ys: Math.random() * 10 + 10
    }))
  }

  _clearRainDrops () {
    let { canvas, ctx } = this.rainDrops
    ctx.clearRect(0, 0, canvas.width, canvas.height)
  }

  _drawRainDrop (i) {
    let { ctx, data } = this.rainDrops

    ctx.beginPath()
    ctx.moveTo(data[i].x, data[i].y)
    ctx.lineTo(data[i].x + data[i].l * data[i].xs, data[i].y + data[i].l * data[i].ys)
    ctx.strokeStyle = 'rgba(174,194,224,0.5)'
    ctx.lineWidth = 1
    ctx.lineCap = 'round'
    ctx.stroke()
  }

  _animateRainDrops () {
    this._clearRainDrops()

    let { canvas, data } = this.rainDrops

    for (let i = 0; i < this.opts.rainDrops; i++) {
      data[i].x += data[i].xs
      data[i].y += data[i].ys

      if (data[i].x > canvas.width || data[i].y > canvas.height) {
        data[i].x = Math.random() * canvas.width
        data[i].y = -20
      }

      this._drawRainDrop(i)
    }
  }

  _initLightening () {
    this.lightening.current = 0
    this.lightening.total = 0
    this.lightening.playing = false
  }

  _thunder () {
    if (this.lightening.playing) return
    this.lightening.playing = true
    let audio = new window.Audio(`${src}/assets/strike-${Math.floor(random(1, 3))}.mp3`)
    audio.play()
    setTimeout(() => {
      this.lightening.playing = false
    }, 3000) // 3 seconds seems to be a good amount of time instead of 'ended'
  }

  _clearLightening () {
    let { ctx, canvas } = this.lightening
    ctx.globalCompositeOperation = 'destination-out'
    ctx.fillStyle = 'rgba(0,0,0,' + random(1, 30) / 100 + ')'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    ctx.globalCompositeOperation = 'source-over'
  }

  _createLightning () {
    let { data, canvas } = this.lightening

    let x = random(100, canvas.width - 100)
    let y = random(0, canvas.height / 4)
    let createCount = random(1, 3)

    for (var i = 0; i < createCount; i++) {
      let single = {
        x: x,
        y: y,
        xRange: random(5, 30),
        yRange: random(10, 25),
        path: [{
          x: x,
          y: y
        }],
        pathLimit: random(40, 55)
      }

      data.push(single)
    }
  }

  _drawLightning () {
    let { data, ctx, canvas } = this.lightening

    for (var i = 0; i < data.length; i++) {
      var light = data[i]

      light.path.push({
        x: light.path[light.path.length - 1].x + (random(0, light.xRange) - (light.xRange / 2)),
        y: light.path[light.path.length - 1].y + (random(0, light.yRange))
      })

      if (light.path.length > light.pathLimit) {
        data.splice(i, 1)
      }

      ctx.strokeStyle = 'rgba(255, 255, 255, .1)'
      ctx.lineWidth = 3
      if (random(0, 15) === 0) {
        ctx.lineWidth = 6
      }
      if (random(0, 30) === 0) {
        ctx.lineWidth = 8
      }

      ctx.beginPath()
      ctx.moveTo(light.x, light.y)
      for (var pc = 0; pc < light.path.length; pc++) {
        ctx.lineTo(light.path[pc].x, light.path[pc].y)
      }
      if (Math.floor(random(0, 30)) === 1) {
        ctx.fillStyle = 'rgba(255, 255, 255, ' + random(1, 3) / 100 + ')'
        ctx.fillRect(0, 0, canvas.width, canvas.height)
      }
      ctx.lineJoin = 'miter'
      ctx.stroke()
    }
  };

  _animateLightning () {
    this._clearLightening()
    this.lightening.current++

    if (this.lightening.current >= this.lightening.total) {
      this._createLightning()
      this._thunder()
      this.lightening.current = 0
      this.lightening.total = random(100, 200)
    }

    this._drawLightning()
  }

  start () {
    this._animateRainTrough()
    this._animateRainDrops()
    this._animateLightning()

    window.requestAnimationFrame(this.start.bind(this))
  }
}

const storm = new Thunderstorm(document.getElementById('thunderstorm'))

storm.start()
