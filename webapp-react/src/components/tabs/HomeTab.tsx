import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import CountdownTimer from '../common/CountdownTimer'
import RSVPForm from '../RSVPForm'
import { loadConfig } from '../../utils/api'
import { useRegistration } from '../../contexts/RegistrationContext'
import type { Config } from '../../types'

export default function HomeTab() {
  const [config, setConfig] = useState<Config | null>(null)
  const { isRegistered, isLoading: registrationLoading, refreshRegistration } = useRegistration()
  const [formSubmitted, setFormSubmitted] = useState(false)

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  const handleFormSuccess = () => {
    setFormSubmitted(true)
    refreshRegistration() // Обновляем статус регистрации
  }

  return (
    <div className="min-h-screen">
      {/* Hero Section */}
      <section className="relative w-full">
        <div className="relative h-[60vh] min-h-[400px] overflow-hidden">
          {/* Размытый фон */}
          <img
            src="/welcome_photo.jpeg"
            alt="Валентин и Мария"
            className="w-full h-full object-cover object-[center_top] blur-md scale-110"
            onError={(e) => {
              ;(e.target as HTMLImageElement).style.display = 'none'
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
                    ;(e.target as HTMLImageElement).style.display = 'none'
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
      {!registrationLoading && !isRegistered && !formSubmitted && (
        <section className="px-4 pt-4 pb-0">
          <SectionCard className="rsvp-section">
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
          <VenueInfo />
        </SectionCard>
        <VenueMap />
      </section>
    </div>
  )
}

function VenueInfo() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5 }}
      className="text-center mb-3"
    >
      <h3 className="text-2xl md:text-3xl font-secondary font-semibold text-primary mb-1 leading-[1.2]">
        Токсово
      </h3>
      <p className="text-lg md:text-xl font-main text-gray-700 mb-1 leading-[1.2]">Панорама Холл</p>
      <p className="text-[16.8px] md:text-[19.2px] text-gray-600 leading-[1.2]">
        Разъезжая улица, 15, городской посёлок Токсово, Токсовское городское поселение,
        Всеволожский район, Ленинградская область
      </p>
    </motion.div>
  )
}

function VenueMap() {
  const lat = 60.136143
  const lon = 30.525849
  const zoom = 15

  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5 }}
      className="w-full aspect-video rounded-lg overflow-hidden shadow-lg mb-4"
    >
      <iframe
        src={`https://yandex.ru/map-widget/v1/?ll=${lon},${lat}&z=${zoom}&pt=${lon},${lat}`}
        width="100%"
        height="100%"
        frameBorder="0"
        className="border-0"
        allowFullScreen
      />
    </motion.div>
  )
}

