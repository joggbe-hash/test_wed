export type UnauthorizedHandler = () => void | Promise<void>

let unauthorizedHandler: UnauthorizedHandler | null = null

export function setUnauthorizedHandler(handler: UnauthorizedHandler | null) {
  unauthorizedHandler = handler
}

export function notifyUnauthorized() {
  return unauthorizedHandler?.()
}
