import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import { loadConfig, submitRSVP, cancelInvitation } from '../../utils/api'
import { showAlert, showConfirm, hapticFeedback } from '../../utils/telegram'
import type { Config, Guest } from '../../types'

const MAX_GUESTS = 9

export default function MenuTab() {
  const [config, setConfig] = useState<Config | null>(null)
  const [guests, setGuests] = useState<Guest[]>([])
  const [formData, setFormData] = useState({
    lastName: '',
    firstName: '',
    category: '',
    side: '',
  })
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isSuccess, setIsSuccess] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  const addGuest = () => {
    if (guests.length >= MAX_GUESTS) {
      showAlert(`Можно добавить максимум ${MAX_GUESTS} гостей`)
      hapticFeedback('medium')
      return
    }

    const newGuest: Guest = {
      id: Date.now(),
      firstName: '',
      lastName: '',
      telegram: '',
    }
    setGuests([...guests, newGuest])
    hapticFeedback('light')
  }

  const removeGuest = (id: number) => {
    setGuests(guests.filter((g) => g.id !== id))
    hapticFeedback('light')
  }

  const updateGuest = (id: number, field: keyof Guest, value: string) => {
    setGuests(
      guests.map((g) => (g.id === id ? { ...g, [field]: value } : g))
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (!formData.lastName || !formData.firstName || !formData.category || !formData.side) {
      setError('Заполните все обязательные поля')
      hapticFeedback('medium')
      return
    }

    setIsSubmitting(true)
    hapticFeedback('medium')

    const result = await submitRSVP({
      lastName: formData.lastName,
      firstName: formData.firstName,
      category: formData.category,
      side: formData.side,
      guests: guests.map((g) => ({
        firstName: g.firstName,
        lastName: g.lastName,
        telegram: g.telegram,
      })),
    })

    setIsSubmitting(false)

    if (result.success) {
      setIsSuccess(true)
      hapticFeedback('heavy')
      // Очищаем форму
      setFormData({ lastName: '', firstName: '', category: '', side: '' })
      setGuests([])
    } else {
      setError(result.error || 'Ошибка регистрации')
      hapticFeedback('heavy')
    }
  }

  const handleCancel = async () => {
    const confirmed = await showConfirm('Вы уверены, что хотите отменить приглашение?')
    if (!confirmed) return

    hapticFeedback('medium')
    const result = await cancelInvitation()

    if (result.success) {
      showAlert('Приглашение отменено. Вы можете заполнить форму заново.')
      setIsSuccess(false)
      setFormData({ lastName: '', firstName: '', category: '', side: '' })
      setGuests([])
    } else {
      showAlert(result.error || 'Ошибка при отмене приглашения')
    }
  }

  return (
    <div className="min-h-screen px-4 py-4">
      {/* RSVP Form */}
      <section className="rsvp-section">
        <SectionCard>
        <AnimatePresence>
          {isSuccess ? (
            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9 }}
              className="text-center py-4"
            >
              <SectionTitle>Спасибо!</SectionTitle>
              <p className="text-[21.6px] text-gray-700 mb-3 leading-[1.2]">
                Мы ждем вас на нашей свадьбе! 💕
              </p>
            </motion.div>
          ) : (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
            >
              <SectionTitle>ПРИСУТСТВИЕ</SectionTitle>
              <p className="text-center text-gray-600 mb-2 leading-[1.2] text-[19.2px]">
                Пожалуйста, подтвердите ваше присутствие на нашем празднике.
                Заполните форму ниже:
              </p>

              {error && (
                <motion.div
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="bg-red-50 border-2 border-red-200 rounded-lg p-2 mb-2 text-red-700"
                >
                  ⚠️ {error}
                </motion.div>
              )}

              <form onSubmit={handleSubmit} className="rsvp-form">
                {/* Main Guest */}
                <div className="form-group">
                  <label htmlFor="lastName">Фамилия, Имя</label>
                  <input
                    type="text"
                    id="lastName"
                    required
                    minLength={2}
                    placeholder="Фамилия"
                    value={formData.lastName}
                    onChange={(e) =>
                      setFormData({ ...formData, lastName: e.target.value })
                    }
                  />
                  <input
                    type="text"
                    id="firstName"
                    required
                    minLength={2}
                    placeholder="Имя"
                    value={formData.firstName}
                    onChange={(e) =>
                      setFormData({ ...formData, firstName: e.target.value })
                    }
                  />
                </div>

                <div className="form-group">
                  <label htmlFor="category">Родство</label>
                  <select
                    id="category"
                    required
                    value={formData.category}
                    onChange={(e) =>
                      setFormData({ ...formData, category: e.target.value })
                    }
                    className="form-select"
                  >
                    <option value="">Выберите...</option>
                    <option value="Семья">Семья</option>
                    <option value="Друзья">Друзья</option>
                    <option value="Родственники">Родственники</option>
                  </select>
                </div>

                <div className="form-group">
                  <label htmlFor="side">Сторона</label>
                  <select
                    id="side"
                    required
                    value={formData.side}
                    onChange={(e) =>
                      setFormData({ ...formData, side: e.target.value })
                    }
                    className="form-select"
                  >
                    <option value="">Выберите...</option>
                    <option value="Жених">Жених</option>
                    <option value="Невеста">Невеста</option>
                    <option value="Общие">Общие</option>
                  </select>
                </div>

                {/* Additional Guests */}
                <div className="guests-list">
                  <AnimatePresence>
                    {guests.map((guest) => (
                      <GuestForm
                        key={guest.id}
                        guest={guest}
                        onUpdate={(field, value) =>
                          updateGuest(guest.id, field, value)
                        }
                        onRemove={() => removeGuest(guest.id)}
                      />
                    ))}
                  </AnimatePresence>
                </div>

                <div className="form-group">
                  <button
                    type="button"
                    onClick={addGuest}
                    className="btn-add-guest"
                  >
                    + Добавить гостя
                  </button>
                </div>

                <div className="form-buttons">
                  <button
                    type="submit"
                    disabled={isSubmitting}
                    className="btn-confirm"
                  >
                    {isSubmitting ? 'Отправка...' : 'Подтвердить'}
                  </button>
                </div>
              </form>
            </motion.div>
          )}
        </AnimatePresence>
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
        <div className="mt-3 text-center">
          <button
            onClick={handleCancel}
            className="px-[4.8px] py-[1.6px] text-red-600 border-2 border-red-300 rounded-lg hover:bg-red-50 transition-colors"
          >
            Отменить приглашение
          </button>
        </div>
      </SectionCard>
    </div>
  )
}

function GuestForm({
  guest,
  onUpdate,
  onRemove,
}: {
  guest: Guest
  onUpdate: (field: keyof Guest, value: string) => void
  onRemove: () => void
}) {
  return (
    <motion.div
      initial={{ opacity: 0, x: -20 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 20 }}
      className="guest-item"
    >
      <div className="flex justify-between items-center mb-1">
        <h4 className="font-semibold text-gray-700">Гость</h4>
        <button
          type="button"
          onClick={onRemove}
          className="btn-remove"
        >
          Удалить
        </button>
      </div>
      <div className="form-group">
        <div className="grid grid-cols-2 gap-1.5">
          <input
            type="text"
            placeholder="Фамилия"
            value={guest.lastName}
            onChange={(e) => onUpdate('lastName', e.target.value)}
            required
          />
          <input
            type="text"
            placeholder="Имя"
            value={guest.firstName}
            onChange={(e) => onUpdate('firstName', e.target.value)}
            required
          />
        </div>
        <input
          type="text"
          placeholder="Telegram (необязательно)"
          value={guest.telegram || ''}
          onChange={(e) => onUpdate('telegram', e.target.value)}
        />
      </div>
    </motion.div>
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

