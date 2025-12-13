export const tg = window.Telegram?.WebApp

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
  return tg?.initData || ''
}

