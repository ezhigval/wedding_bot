import { useEffect, useState, useCallback } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'
import { useUser } from '../../contexts/UserContext'
import {
  getPersonalSeatingInfo,
  getSeatingInfo,
  type PersonalSeatingInfo,
  type SeatingInfo,
  type SeatingTable,
} from '../../utils/api'

type SeatingViewMode = 'personal' | 'public'

function formatPublishedAt(raw?: string): string | null {
  if (!raw) {
    return null
  }

  const normalized = raw.replace(' ', 'T')
  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) {
    return raw
  }

  return date.toLocaleString('ru-RU', {
    day: '2-digit',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function SeatingModeButton({
  active,
  label,
  onClick,
}: {
  active: boolean
  label: string
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={`rounded-lg px-4 py-3 text-[16.8px] font-semibold transition-all ${
        active
          ? 'bg-primary text-white shadow-md'
          : 'bg-cream/30 text-primary-dark hover:bg-cream/50'
      }`}
    >
      {label}
    </button>
  )
}

function SeatingTableCard({ table, index }: { table: SeatingTable; index: number }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 16 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.35, delay: index * 0.04 }}
      className="overflow-hidden rounded-lg border border-primary/15 bg-white/90 shadow-sm"
    >
      <div className="bg-gradient-to-r from-primary to-primary-dark px-4 py-3 text-white">
        <div className="flex items-end justify-between gap-3">
          <div>
            <div className="text-[14px] uppercase tracking-wide text-white/80">Стол</div>
            <div className="text-[26px] font-secondary font-bold leading-none">{table.table}</div>
          </div>
          <div className="rounded-full bg-white/15 px-3 py-1 text-[14px] font-semibold text-white">
            {table.guests.length} гостей
          </div>
        </div>
      </div>

      <div className="space-y-2 px-4 py-4">
        {table.guests.map((guest, guestIndex) => (
          <div
            key={`${table.table}-${guestIndex}-${guest}`}
            className="flex items-start gap-3 rounded-lg bg-cream/30 px-3 py-2 text-[18px] text-gray-700"
          >
            <span className="mt-0.5 inline-flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-primary/10 text-[14px] font-semibold text-primary-dark">
              {guestIndex + 1}
            </span>
            <span className="leading-[1.25]">{guest}</span>
          </div>
        ))}
      </div>
    </motion.div>
  )
}

function RetryBlock({ title, message, onRetry }: { title: string; message: string; onRetry: () => void }) {
  return (
    <div className="text-center py-4">
      <p className="mb-3 text-[19.2px] text-red-600">{title}</p>
      <p className="mb-4 text-[16.8px] text-gray-500">{message}</p>
      <button
        onClick={onRetry}
        className="rounded-lg bg-primary px-4 py-2 font-semibold text-white transition-colors hover:bg-primary-dark"
      >
        Повторить
      </button>
    </div>
  )
}

function PersonalSeatingView({
  publicInfo,
  personalInfo,
  onRetry,
}: {
  publicInfo: SeatingInfo | null
  personalInfo: PersonalSeatingInfo | null
  onRetry: () => void
}) {
  if (!publicInfo?.visible) {
    return (
      <div className="text-center py-4">
        <p className="mb-2 text-[19.2px] text-gray-700">Рассадка ещё не опубликована.</p>
        <p className="text-[16.8px] text-gray-500">
          Когда организаторы нажмут «Обновить рассадку» в админ-панели, здесь появится ваш стол.
        </p>
      </div>
    )
  }

  if (personalInfo?.error) {
    return (
      <RetryBlock
        title="Не удалось загрузить персональную рассадку."
        message={personalInfo.error}
        onRetry={onRetry}
      />
    )
  }

  if (!personalInfo?.visible) {
    return (
      <div className="text-center py-4">
        <p className="mb-2 text-[19.2px] text-gray-700">Ваш стол пока не найден.</p>
        <p className="text-[16.8px] text-gray-500">
          Общая рассадка уже доступна. Если ваше место не отображается и здесь, попросите организаторов проверить список гостей и опубликованную рассадку.
        </p>
      </div>
    )
  }

  const neighbors = personalInfo.neighbors || []

  return (
    <div className="space-y-3">
      {personalInfo.full_name && (
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.3 }}
          className="rounded-lg border border-primary/20 bg-cream/40 px-4 py-3 text-center"
        >
          <div className="text-[16.8px] uppercase tracking-wide text-primary/80">Гость</div>
          <div className="text-[24px] font-semibold text-primary-dark">{personalInfo.full_name}</div>
        </motion.div>
      )}

      <motion.div
        initial={{ opacity: 0, y: 12 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.35 }}
        className="rounded-lg bg-primary text-white px-4 py-4 text-center shadow-md"
      >
        <div className="text-[16.8px] uppercase tracking-wide text-white/80">Ваш стол</div>
        <div className="mt-1 text-[38px] font-secondary font-bold leading-none">
          {personalInfo.table || 'Без номера'}
        </div>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 12 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.4 }}
        className="rounded-lg border border-primary/15 bg-white/80 px-4 py-4"
      >
        <h3 className="mb-3 text-center text-[21.6px] font-semibold text-primary-dark">
          Соседи по столу
        </h3>

        {neighbors.length === 0 ? (
          <p className="text-center text-[16.8px] text-gray-500">
            Список соседей появится после финальной публикации рассадки.
          </p>
        ) : (
          <div className="grid gap-2">
            {neighbors.map((neighbor, index) => (
              <div
                key={`${neighbor}-${index}`}
                className="rounded-lg bg-cream/40 px-3 py-2 text-center text-[18px] text-gray-700"
              >
                {neighbor}
              </div>
            ))}
          </div>
        )}
      </motion.div>
    </div>
  )
}

function PublicSeatingView({ publicInfo, onRetry }: { publicInfo: SeatingInfo | null; onRetry: () => void }) {
  if (publicInfo?.error) {
    return (
      <RetryBlock
        title="Не удалось загрузить опубликованную рассадку."
        message={publicInfo.error}
        onRetry={onRetry}
      />
    )
  }

  if (!publicInfo?.visible || publicInfo.tables.length === 0) {
    return (
      <div className="text-center py-4">
        <p className="mb-2 text-[19.2px] text-gray-700">Рассадка ещё не опубликована.</p>
        <p className="text-[16.8px] text-gray-500">
          Когда организаторы нажмут «Обновить рассадку» в админ-панели, список столов появится здесь.
        </p>
      </div>
    )
  }

  return (
    <div className="grid gap-3">
      {publicInfo.tables.map((table, index) => (
        <SeatingTableCard key={table.table || `table-${index}`} table={table} index={index} />
      ))}
    </div>
  )
}

export default function SeatingTab() {
  const { isRegistered, isLoading } = useRegistration()
  const { userId, manualUsername } = useUser()
  const [viewMode, setViewMode] = useState<SeatingViewMode>('personal')
  const [publicInfo, setPublicInfo] = useState<SeatingInfo | null>(null)
  const [personalInfo, setPersonalInfo] = useState<PersonalSeatingInfo | null>(null)
  const [loadingInfo, setLoadingInfo] = useState(true)

  const loadSeating = useCallback(async () => {
    setLoadingInfo(true)
    const [loadedPublicInfo, loadedPersonalInfo] = await Promise.all([
      getSeatingInfo(),
      getPersonalSeatingInfo({ userId, username: manualUsername }),
    ])
    setPublicInfo(loadedPublicInfo)
    setPersonalInfo(loadedPersonalInfo)
    setLoadingInfo(false)
  }, [manualUsername, userId])

  useEffect(() => {
    if (isRegistered) {
      void loadSeating()
    } else {
      setPublicInfo(null)
      setPersonalInfo(null)
      setLoadingInfo(false)
    }
  }, [isRegistered, loadSeating])

  if (isLoading) {
    return (
      <div className="min-h-screen px-4 py-4 flex items-center justify-center">
        <div className="text-center text-gray-500">Загрузка...</div>
      </div>
    )
  }

  if (!isRegistered) {
    return <RegistrationRequired />
  }

  if (loadingInfo) {
    return (
      <div className="min-h-screen px-4 py-4 flex items-center justify-center">
        <div className="text-center text-gray-500">Загружаем рассадку...</div>
      </div>
    )
  }

  const publishedAt = formatPublishedAt(personalInfo?.published_at || publicInfo?.published_at)

  return (
    <div className="min-h-screen px-4 py-4 pb-[calc(env(safe-area-inset-bottom,0px)+112px)]">
      <SectionCard>
        <SectionTitle>РАССАДКА</SectionTitle>

        <div className="mb-4 grid grid-cols-2 gap-2">
          <SeatingModeButton active={viewMode === 'personal'} label="Моя рассадка" onClick={() => setViewMode('personal')} />
          <SeatingModeButton active={viewMode === 'public'} label="Общая рассадка" onClick={() => setViewMode('public')} />
        </div>

        <div className="mb-4 rounded-lg border border-primary/15 bg-cream/30 px-4 py-3 text-center text-gray-700">
          <div className="text-[16.8px] uppercase tracking-wide text-primary/80">Опубликованная версия</div>
          <div className="mt-1 text-[18px] font-semibold text-primary-dark">
            {publishedAt ? `Обновлено ${publishedAt}` : 'Актуальная рассадка из админ-панели'}
          </div>
        </div>

        {viewMode === 'personal' ? (
          <PersonalSeatingView publicInfo={publicInfo} personalInfo={personalInfo} onRetry={() => void loadSeating()} />
        ) : (
          <PublicSeatingView publicInfo={publicInfo} onRetry={() => void loadSeating()} />
        )}
      </SectionCard>
    </div>
  )
}
