function mf2ToInternal (data) {
  if (typeof data !== 'object') {
    return data
  }

  if (Array.isArray(data)) {
    return data.length === 1
      ? mf2ToInternal(data[0])
      : data.map(mf2ToInternal)
  }

  const parsed = {}

  for (const [key, value] of Object.entries(data)) {
    parsed[key] = mf2ToInternal(value)
  }

  return parsed
}

function internalToMf2 (data) {
  if (data === null) {
    return [null]
  }

  if (typeof data !== 'object') {
    return data
  }

  if (Array.isArray(data)) {
    return data.map(internalToMf2)
  }

  const parsed = {}

  for (const [key, value] of Object.entries(data)) {
    parsed[key] = Array.isArray(value) || key === 'properties' || key === 'value'
      ? internalToMf2(value)
      : [internalToMf2(value)]
  }

  return parsed
}

module.exports = {
  internalToMf2,
  mf2ToInternal
}
