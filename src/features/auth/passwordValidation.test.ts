import { describe, expect, it } from 'vitest'
import { registrationPasswordError } from './passwordValidation'

describe('registrationPasswordError', () => {
  it('accepts passwords with at least eight Unicode characters, a letter, and a number', () => {
    expect(registrationPasswordError('Password1')).toBe('')
    expect(registrationPasswordError('密碼安全測試七8號')).toBe('')
  })

  it('reports a password shorter than eight Unicode characters', () => {
    expect(registrationPasswordError('密碼a123')).toBe('密碼至少需要 8 個字元')
  })

  it('reports the bcrypt byte limit separately', () => {
    expect(registrationPasswordError(`${'密'.repeat(24)}1`)).toBe('密碼過長，請縮短後再試')
  })

  it('reports missing letters or numbers', () => {
    expect(registrationPasswordError('abcdefgh')).toBe('密碼須包含至少一個字母與一個數字')
    expect(registrationPasswordError('12345678')).toBe('密碼須包含至少一個字母與一個數字')
  })
})
