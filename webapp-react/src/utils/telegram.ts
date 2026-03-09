export const tg = window.Telegram?.WebApp

function getUrlParam(name: string): string | null {
  try {
    const url = new URL(window.location.href)
    const direct = url.searchParams.get(name)
    if (direct) return direct

    // Telegram иногда кладёт параметры в hash
    const hash = url.hash.startsWith('#') ? url.hash.slice(1) : url.hash
    if (!hash) return null
    const hashParams = new URLSearchParams(hash)
    return hashParams.get(name)
  } catch {
    return null
  }
}

export function getInitDataAny(): string {
  // 1) Внутри Telegram WebApp
  const initData = tg?.initData
  if (initData) return initData

  // 2) При открытии не внутри Telegram, но с параметром (например, tgWebAppData)
  const tgWebAppData = getUrlParam('tgWebAppData')
  if (tgWebAppData) {
    try {
      return decodeURIComponent(tgWebAppData)
    } catch {
      return tgWebAppData
    }
  }

  const initDataParam = getUrlParam('initData')
  if (initDataParam) {
    try {
      return decodeURIComponent(initDataParam)
    } catch {
      return initDataParam
    }
  }

  return ''
}

export function hapticFeedback(style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft' = 'light') {
  tg?.HapticFeedback?.impactOccurred(style)
}

export function showAlert(message: string) {
  tg?.showAlert(message)
}

export async function showConfirm(message: string): Promise<boolean> {
  if (tg?.showConfirm) {
    return tg.showConfirm(message)
  }
  return Promise.resolve(window.confirm(message))
}

export function getInitData(): string {
  return getInitDataAny()
}

