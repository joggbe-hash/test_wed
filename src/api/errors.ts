import { ApiError } from './backendApi'

const statusMessages: Partial<Record<number, string>> = {
  400: '輸入資料不正確，請檢查後再試一次。',
  401: '登入狀態已失效，請重新登入。',
  403: '你沒有權限執行這個操作。',
  404: '找不到指定的資料。',
  409: '資料已存在或發生衝突，請確認後再試。',
  413: '上傳內容超過大小限制。',
  429: '操作太頻繁，請稍後再試。',
  500: '伺服器發生錯誤，請稍後再試。',
  503: '服務目前忙碌中，請稍後再試。',
}

export function apiErrorMessage(error: unknown, fallback = '操作失敗，請稍後再試。') {
  if (error instanceof ApiError) {
    return statusMessages[error.status] ?? fallback
  }
  if (error instanceof TypeError) {
    return '無法連線到伺服器，請檢查網路後再試。'
  }
  return fallback
}

export function loginErrorMessage(error: unknown) {
  if (error instanceof ApiError && error.status === 401) {
    return '電子信箱或密碼不正確'
  }
  if (error instanceof ApiError && error.status === 429) {
    return '登入嘗試次數過多，請稍後再試（最長約 5 分鐘）。'
  }

  return apiErrorMessage(error, '登入失敗，請稍後再試。')
}

export function verificationSendErrorMessage(error: unknown) {
  if (error instanceof ApiError && error.status === 429) {
    return '驗證碼傳送太頻繁。每次需間隔 60 秒；若已達寄送上限，請稍後再試。'
  }

  return apiErrorMessage(error, '驗證碼傳送失敗，請稍後再試。')
}

export function registrationErrorMessage(error: unknown) {
  if (error instanceof ApiError && error.status === 400) {
    return '驗證碼不正確、已過期，或不是最新一封，請確認後再試。'
  }
  if (error instanceof ApiError && error.status === 429) {
    return '驗證碼錯誤次數過多，請使用最新一封的正確驗證碼，或重新取得驗證碼。'
  }

  return apiErrorMessage(error, '註冊失敗，請稍後再試。')
}
