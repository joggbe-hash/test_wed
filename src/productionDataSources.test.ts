import { describe, expect, it } from 'vitest'

const productionSources = import.meta.glob<string>('./**/*.{ts,vue}', {
  eager: true,
  import: 'default',
  query: '?raw',
})
const forbiddenProductionPatterns = [
  /VITE_USE_MOCK_API/,
  /(?:^|[/\\])timedApi(?:\.ts)?/,
  /(?:^|[/\\])fixtures[/\\]demoData(?:\.ts)?/,
  /useScheduleMock/,
]

describe('production data sources', () => {
  it('contains no runtime mock or demo data path', () => {
    const violations = Object.entries(productionSources).flatMap(([path, source]) => {
      if (path.includes('.test.')) return []
      return forbiddenProductionPatterns
        .filter((pattern) => pattern.test(source))
        .map((pattern) => `${path}: ${pattern.source}`)
    })

    expect(violations).toEqual([])
    expect(Object.keys(productionSources)).not.toContain('./api/timedApi.ts')
    expect(Object.keys(productionSources)).not.toContain('./fixtures/demoData.ts')
  })
})
