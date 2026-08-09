const MIN_PASSWORD_CHARACTERS = 8
const MAX_PASSWORD_BYTES = 72

export function registrationPasswordError(password: string): string {
  if (Array.from(password).length < MIN_PASSWORD_CHARACTERS) {
    return '密碼至少需要 8 個字元'
  }

  if (new TextEncoder().encode(password).length > MAX_PASSWORD_BYTES) {
    return '密碼過長，請縮短後再試'
  }

  if (!/\p{L}/u.test(password) || !/\p{N}/u.test(password)) {
    return '密碼須包含至少一個字母與一個數字'
  }

  return ''
}
