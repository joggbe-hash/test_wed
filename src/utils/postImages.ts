export const maxPostImages = 4
export const maxPostImageBytes = 8 * 1024 * 1024
export const acceptedPostImageTypes = ['image/jpeg', 'image/png'] as const
export const acceptedPostImageInput = acceptedPostImageTypes.join(',')

export interface PostImageValidationResult {
  accepted: File[]
  errorMessage: string
}

export function validatePostImages(files: Iterable<File>): PostImageValidationResult {
  const selected = Array.from(files)

  if (selected.length > maxPostImages) {
    return {
      accepted: [],
      errorMessage: `一次最多只能選擇 ${maxPostImages} 張圖片。`,
    }
  }

  const unsupported = selected.find((file) =>
    !acceptedPostImageTypes.includes(file.type as (typeof acceptedPostImageTypes)[number]),
  )
  if (unsupported) {
    return {
      accepted: [],
      errorMessage: `「${unsupported.name}」不是支援的 JPEG 或 PNG 圖片。`,
    }
  }

  const oversized = selected.find((file) => file.size > maxPostImageBytes)
  if (oversized) {
    return {
      accepted: [],
      errorMessage: `「${oversized.name}」超過每張 8 MiB 的限制。`,
    }
  }

  return { accepted: selected, errorMessage: '' }
}

