import { fetchFeed } from './backendApi'
import type { FeedResponse, ImageStatus } from './contracts'

const defaultPollDelaysMs = [1_000, 2_000, 3_000, 5_000, 8_000]

interface WaitForPostProcessingOptions {
  signal?: AbortSignal
  pollDelaysMs?: number[]
  fetcher?: () => Promise<FeedResponse>
}

function wait(delayMs: number, signal?: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException('Polling aborted', 'AbortError'))
      return
    }

    const onAbort = () => {
      window.clearTimeout(timeoutId)
      reject(new DOMException('Polling aborted', 'AbortError'))
    }
    const timeoutId = window.setTimeout(() => {
      signal?.removeEventListener('abort', onAbort)
      resolve()
    }, delayMs)
    signal?.addEventListener('abort', onAbort, { once: true })
  })
}

export async function waitForPostProcessing(
  postId: number,
  options: WaitForPostProcessingOptions = {},
): Promise<ImageStatus | null> {
  const {
    signal,
    pollDelaysMs = defaultPollDelaysMs,
    fetcher = () => fetchFeed(),
  } = options

  for (const delayMs of pollDelaysMs) {
    await wait(delayMs, signal)
    const response = await fetcher()
    const post = response.posts.find((candidate) => candidate.id === postId)
    if (post && post.image_status !== 'processing') return post.image_status
  }

  return null
}
