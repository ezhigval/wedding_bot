export interface Config {
  weddingDate: string
  groomName: string
  brideName: string
  groomTelegram: string
  brideTelegram: string
  weddingAddress: string
  apiUrl: string
}

export interface TimelineItem {
  time: string
  event: string
}

export interface Guest {
  id: number
  firstName: string
  lastName: string
  telegram?: string
}

export interface VenueInfo {
  name: string
  hall: string
  address: string
  lat: number
  lon: number
}

export type TabName = 'home' | 'challenge' | 'menu' | 'photo' | 'timeline' | 'dresscode' | 'seating' | 'wishes'

