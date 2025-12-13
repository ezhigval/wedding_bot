import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RSVPForm from '../RSVPForm'
import { loadConfig, cancelInvitation } from '../../utils/api'
import { showAlert, showConfirm, hapticFeedback } from '../../utils/telegram'
import { useRegistration } from '../../contexts/RegistrationContext'
import type { Config } from '../../types'

export default function MenuTab() {
  const [config, setConfig] = useState<Config | null>(null)
  const { isRegistered, isLoading: registrationLoading, refreshRegistration } = useRegistration()

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  const handleCancel = async () => {
    const confirmed = await showConfirm('Вы уверены, что хотите отменить приглашение?')
    if (!confirmed) return

    hapticFeedback('medium')
    const result = await cancelInvitation()

    if (result.success) {
      showAlert('Приглашение отменено. Вы можете заполнить форму заново.')
      refreshRegistration() // Обновляем статус регистрации
    } else {
      showAlert(result.error || 'Ошибка при отмене приглашения')
    }
  }

  // Показываем форму только для зарегистрированных пользователей
  if (registrationLoading) {
    return (
      <div className="min-h-screen px-4 py-4 flex items-center justify-center">
        <div className="text-center text-gray-500">Загрузка...</div>
      </div>
    )
  }

  if (!isRegistered) {
    return (
      <div className="min-h-screen px-4 py-4">
        <SectionCard>
          <SectionTitle>МЕНЮ</SectionTitle>
          <p className="text-center text-gray-600 mb-2 leading-[1.2] text-[19.2px]">
            Для доступа к меню необходимо подтвердить ваше присутствие на главной странице.
          </p>
        </SectionCard>
      </div>
    )
  }

  return (
    <div className="min-h-screen px-4 py-4">
      {/* RSVP Form for registered users (guests only) */}
      <section className="rsvp-section">
        <SectionCard className="rsvp-section">
          <RSVPForm mode="guests-only" />
        </SectionCard>
      </section>

      {/* Contact Section */}
      <SectionCard>
        <SectionTitle>СВЯЗАТЬСЯ С НАМИ</SectionTitle>
        <div className="space-y-2">
          <ContactItem
            name="Валентин"
            telegram={config?.groomTelegram || 'ezhigval'}
          />
          {config?.brideTelegram && (
            <ContactItem
              name="Мария"
              telegram={config.brideTelegram}
            />
          )}
        </div>
      {/* Cancel Invitation Button - только для зарегистрированных */}
      <SectionCard>
        <div className="mt-3 text-center">
          <button
            onClick={handleCancel}
            className="px-[4.8px] py-[1.6px] text-red-600 border-2 border-red-300 rounded-lg hover:bg-red-50 transition-colors"
          >
            Отменить приглашение
          </button>
        </div>
      </SectionCard>
      </SectionCard>
    </div>
  )
}

function ContactItem({ name, telegram }: { name: string; telegram: string }) {
  return (
    <motion.a
      href={`https://t.me/${telegram.replace('@', '')}`}
      target="_blank"
      rel="noopener noreferrer"
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      className="flex justify-between items-center p-[3.2px] bg-cream/30 rounded-lg border border-primary/20 hover:bg-cream/50 transition-colors"
    >
      <span className="font-semibold text-gray-700">{name}</span>
      <span className="text-primary">@{telegram.replace('@', '')}</span>
    </motion.a>
  )
}

