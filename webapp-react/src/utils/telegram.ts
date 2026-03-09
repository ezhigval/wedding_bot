export function getTelegramWebApp() {
  return window.Telegram?.WebApp
}

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
  const initData = getTelegramWebApp()?.initData
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
  getTelegramWebApp()?.HapticFeedback?.impactOccurred(style)
}

export function showAlert(message: string) {
  const webApp = getTelegramWebApp()
  if (webApp?.showAlert) {
    webApp.showAlert(message)
    return
  }
  window.alert(message)
}

export async function showConfirm(message: string): Promise<boolean> {
  const webApp = getTelegramWebApp()
  if (webApp?.showConfirm) {
    return webApp.showConfirm(message)
  }
  return Promise.resolve(window.confirm(message))
}

export function getInitData(): string {
  return getInitDataAny()
}
