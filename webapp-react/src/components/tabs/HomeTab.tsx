import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import CountdownTimer from '../common/CountdownTimer'
import RSVPForm from '../RSVPForm'
import { loadConfig } from '../../utils/api'
import { useRegistration } from '../../contexts/RegistrationContext'
import { useUser } from '../../contexts/UserContext'
import type { Config } from '../../types'

const DEFAULT_VENUE_NAME = 'Ресторан "Марсала"'
const DEFAULT_VENUE_ADDRESS = 'Большой проспект Петроградской стороны, 84, Санкт-Петербург'
const DEFAULT_VENUE_LAT = 59.9643641
const DEFAULT_VENUE_LON = 30.3092636

export default function HomeTab() {
  const [config, setConfig] = useState<Config | null>(null)
  const { isRegistered, isLoading: registrationLoading, refreshRegistration } = useRegistration()
  const { userId, manualUsername, setManualUsername, refreshUserId } = useUser()
  const [usernameInput, setUsernameInput] = useState(manualUsername ? `@${manualUsername}` : '')

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  useEffect(() => {
    setUsernameInput(manualUsername ? `@${manualUsername}` : '')
  }, [manualUsername])

  const handleFormSuccess = async () => {
    // После успешной отправки формы принудительно переобновляем identity и регистрацию,
    // чтобы убрать рассинхрон между вкладками и главной.
    await refreshUserId()
    await refreshRegistration()
  }

  const trimmedUsername = usernameInput.trim()

  return (
    <div className="min-h-screen pb-[calc(env(safe-area-inset-bottom,0px)+112px)]">
      {/* Hero Section */}
      <section className="relative w-full">
        <div className="relative h-[60vh] min-h-[400px] overflow-hidden">
          {/* Размытый фон */}
          <img
            src="/welcome_photo.jpeg"
            alt="Валентин и Мария"
            className="w-full h-full object-cover object-[center_top] blur-md scale-110"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none'
            }}
          />
          <div className="absolute inset-0 bg-gradient-to-b from-black/40 via-black/60 to-black/80" />
          
          {/* Контент с фотографией в кружочке */}
          <div className="absolute inset-0 flex flex-col items-center justify-center px-4">
            {/* Фотография в кружочке */}
            <motion.div
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.6 }}
              className="relative mb-6"
            >
              <div className="w-48 h-48 md:w-56 md:h-56 rounded-full overflow-hidden border-4 border-gray-300 shadow-2xl">
                <img
                  src="/welcome_photo.jpeg"
                  alt="Валентин и Мария"
                  className="w-full h-full object-cover object-[center_top]"
                  onError={(e) => {
                    (e.target as HTMLImageElement).style.display = 'none'
                  }}
                />
              </div>
            </motion.div>

            {/* Имена */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.2 }}
              className="text-4xl md:text-5xl font-secondary font-bold mb-2 text-center leading-[1.2] text-white"
            >
              {config ? `${config.groomName} и ${config.brideName}` : 'Валентин и Мария'}
            </motion.div>
            
            {/* Дата */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.4 }}
              className="text-xl md:text-2xl font-main leading-[1.2] text-[#C8A067]"
            >
              {config?.weddingDate
                ? new Date(config.weddingDate).toLocaleDateString('ru-RU', {
                    day: 'numeric',
                    month: 'long',
                    year: 'numeric',
                  })
                : '05 июня 2026'}
            </motion.div>
          </div>
        </div>
      </section>

      {/* Greeting Section */}
      <section className="px-4 pt-4 pb-0">
        <SectionCard>
          <SectionTitle>ДОРОГИЕ РОДНЫЕ И БЛИЗКИЕ</SectionTitle>
          <motion.p
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: 0.2 }}
            className="text-[19.2px] md:text-[21.6px] leading-[1.2] text-gray-700 text-center"
          >
            Дорогие родные и близкие! <br />
            <br />
            Мы давно ждали момента, когда сможем разделить с вами самый важный и счастливый день в
            нашей жизни. Скоро состоится наша свадьба! Мы рады пригласить вас стать свидетелями
            этого торжества и разделить с нами самые яркие моменты!
          </motion.p>
        </SectionCard>
      </section>

      {/* RSVP Form for unregistered users */}
      {!registrationLoading && !isRegistered && (
        <section className="px-4 pt-4 pb-0">
          <SectionCard className="rsvp-section">
            {!userId && (
              <div className="mb-3">
                <SectionTitle>ВХОД</SectionTitle>
                <p className="text-center text-gray-600 mb-2 leading-[1.2] text-[16.8px]">
                  Если вы открыли ссылку вне Telegram, введите ваш Telegram username.
                </p>
                <div className="flex gap-2 items-center justify-center">
                  <input
                    value={usernameInput}
                    onChange={(e) => setUsernameInput(e.target.value)}
                    placeholder="@username"
                    className="border border-gray-300 rounded-lg px-3 py-2 text-[16.8px] w-56"
                  />
                  <button
                    disabled={trimmedUsername === ''}
                    className="bg-primary text-white rounded-lg px-4 py-2 font-semibold disabled:cursor-not-allowed disabled:opacity-50"
                    onClick={async () => {
                      setManualUsername(trimmedUsername)
                      await refreshRegistration({ username: trimmedUsername })
                    }}
                  >
                    Войти
                  </button>
                </div>
              </div>
            )}
            <RSVPForm mode="full" onSuccess={handleFormSuccess} />
          </SectionCard>
        </section>
      )}

      {/* Countdown Timer Section */}
      {config && (
        <CountdownTimer weddingDate={config.weddingDate} />
      )}

      {/* Venue Section */}
      <section className="px-4 py-0">
        <SectionCard>
          <SectionTitle>МЕСТО ПРОВЕДЕНИЯ</SectionTitle>
          <VenueInfo address={config?.weddingAddress} />
        </SectionCard>
        <VenueMap address={config?.weddingAddress} />
      </section>
    </div>
  )
}

function resolveVenue(address?: string) {
  const normalized = (address || '').trim()
  if (!normalized) {
    return {
      title: DEFAULT_VENUE_NAME,
      details: DEFAULT_VENUE_ADDRESS,
      mapQuery: `${DEFAULT_VENUE_NAME}, ${DEFAULT_VENUE_ADDRESS}`,
    }
  }

  const normalizedForMatch = normalized.toLowerCase().replace(/[«»"]/g, '')
  const hasVenueName = normalizedForMatch.includes('марсала')
  const parts = normalized.split(',').map((part) => part.trim()).filter(Boolean)

  if (!hasVenueName) {
    return {
      title: DEFAULT_VENUE_NAME,
      details: normalized,
      mapQuery: `${DEFAULT_VENUE_NAME}, ${normalized}`,
    }
  }

  if (parts.length === 0) {
    return {
      title: DEFAULT_VENUE_NAME,
      details: DEFAULT_VENUE_ADDRESS,
      mapQuery: `${DEFAULT_VENUE_NAME}, ${DEFAULT_VENUE_ADDRESS}`,
    }
  }

  const title = parts[0] || DEFAULT_VENUE_NAME
  const details = parts.slice(1).join(', ') || DEFAULT_VENUE_ADDRESS

  return {
    title,
    details,
    mapQuery: `${title}, ${details}`,
  }
}

function VenueInfo({ address }: { address?: string }) {
  const venue = resolveVenue(address)

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5 }}
      className="text-center mb-3"
    >
      <h3 className="text-2xl md:text-3xl font-secondary font-semibold text-primary mb-1 leading-[1.2]">
        {venue.title}
      </h3>
      {venue.details !== venue.title && (
        <p className="text-[16.8px] md:text-[19.2px] text-gray-600 leading-[1.2]">
          {venue.details}
        </p>
      )}
    </motion.div>
  )
}

function VenueMap({ address }: { address?: string }) {
  const venue = resolveVenue(address)
  const query = encodeURIComponent(venue.mapQuery)
  const normalizedQuery = venue.mapQuery.toLowerCase()
  const isMarsalaVenue = normalizedQuery.includes('марсала')
  const mapUrl = isMarsalaVenue
    ? `https://yandex.ru/map-widget/v1/?ll=${DEFAULT_VENUE_LON},${DEFAULT_VENUE_LAT}&z=16&pt=${DEFAULT_VENUE_LON},${DEFAULT_VENUE_LAT},pm2rdm&mode=search&text=${query}`
    : `https://yandex.ru/map-widget/v1/?mode=search&text=${query}&z=16`
  const directionsUrl = `https://yandex.ru/maps/?mode=search&text=${query}`

  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5 }}
      className="mb-4 space-y-3"
    >
      <div className="w-full aspect-video rounded-lg overflow-hidden shadow-lg">
        <iframe
          src={mapUrl}
          width="100%"
          height="100%"
          frameBorder="0"
          className="border-0"
          allowFullScreen
          loading="lazy"
          referrerPolicy="no-referrer-when-downgrade"
          title="Карта ресторана Марсала"
        />
      </div>
      <a
        href={directionsUrl}
        target="_blank"
        rel="noreferrer"
        className="block rounded-lg border border-[#E5C98B] bg-[#FFF7E2] px-4 py-3 text-center text-[16.8px] font-semibold text-primary transition-colors hover:bg-[#FBEBC1]"
      >
        Открыть в Яндекс Картах
      </a>
    </motion.div>
  )
}
