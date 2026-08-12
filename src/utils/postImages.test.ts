import { describe, expect, it } from 'vitest'
import {
  maxPostImageBytes,
  validatePostImages,
} from './postImages'

function image(name: string, type: string, size = 1) {
  return new File([new Uint8Array(size)], name, { type })
}

describe('validatePostImages', () => {
  it('accepts up to four JPEG or PNG images within the size limit', () => {
    const files = [
      image('one.jpg', 'image/jpeg'),
      image('two.png', 'image/png'),
    ]

    expect(validatePostImages(files)).toEqual({ accepted: files, errorMessage: '' })
  })

  it('rejects unsupported formats', () => {
    const result = validatePostImages([image('animation.gif', 'image/gif')])

    expect(result.accepted).toEqual([])
    expect(result.errorMessage).toContain('JPEG 或 PNG')
  })

  it('rejects files larger than three MiB', () => {
    const result = validatePostImages([
      image('large.png', 'image/png', maxPostImageBytes + 1),
    ])

    expect(result.accepted).toEqual([])
    expect(result.errorMessage).toContain('3 MiB')
  })

  it('rejects more than four files', () => {
    const files = Array.from({ length: 5 }, (_, index) => image(`${index}.png`, 'image/png'))

    expect(validatePostImages(files).errorMessage).toContain('最多')
  })
})
