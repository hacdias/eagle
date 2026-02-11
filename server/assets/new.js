const locationInput = document.querySelector("input[name='location']")
const locationUpdateButton = document.getElementById('location-update-button')
const photosInput = document.getElementById('photos-input')
const photosAddButton = document.getElementById('photos-add-button')
const tagInput = document.getElementById('tag-input')

function updateLocation() {
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      const latitude = Math.round(pos.coords.latitude * 100000) / 100000
      const longitude = Math.round(pos.coords.longitude * 100000) / 100000
      const accuracy = Math.round(pos.coords.accuracy * 100000) / 100000
      const geo = `geo:${latitude},${longitude};u=${accuracy}`
      locationInput.value = geo
    },
    (err) => {
      if (err.code === 1) {
        alert('The website was not able to get permission')
      } else if (err.code === 2) {
        alert('Location information was unavailable')
      } else if (err.code === 3) {
        alert('Timed out getting location')
      }
    },
  )
}

photosAddButton.addEventListener('click', () => {
  photosInput.click()
})

photosInput.addEventListener('change', async () => {
  const files = photosInput.files

  if (files.length !== 1) {
    return
  }

  const formData = new FormData()
  formData.set('file', files[0])

  const response = await fetch('/panel/cache', {
    method: 'POST',
    body: formData,
  })

  if (!response.ok) {
    alert('Failed to upload photo')
    return
  }

  const value = await response.text()

  const input = document.createElement('input')
  input.name = 'photos'
  input.type = 'text'
  input.value = value

  document.getElementById('photos').insertBefore(input, photosAddButton)
})

locationUpdateButton.addEventListener('click', updateLocation)

tagInput.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' || e.key === 'Tab') {
    e.preventDefault()

    const value = e.target.value.trim()

    const input = document.createElement('input')
    input.name = 'tags'
    input.type = 'hidden'
    input.value = value

    const span = document.createElement('span')
    span.textContent = value
    span.addEventListener('click', () => {
      span.remove()
      input.remove()
    })

    document.getElementById('tags').insertBefore(span, e.target)
    document.getElementById('tags').insertBefore(input, e.target)

    e.target.value = ''
  }
})

document.addEventListener('DOMContentLoaded', () => {
  updateLocation()
})
