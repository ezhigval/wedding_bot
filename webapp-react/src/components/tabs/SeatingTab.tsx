import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'
import { getSeatingInfo, type SeatingInfo, type SeatingTable } from '../../utils/api'

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

export default function SeatingTab() {
  const { isRegistered, isLoading } = useRegistration()
  const [seatingInfo, setSeatingInfo] = useState<SeatingInfo | null>(null)
  const [loadingInfo, setLoadingInfo] = useState(true)

  const loadSeating = async () => {
    setLoadingInfo(true)
    const result = await getSeatingInfo()
    setSeatingInfo(result)
    setLoadingInfo(false)
  }

  useEffect(() => {
    if (isRegistered) {
      void loadSeating()
    } else {
      setSeatingInfo(null)
      setLoadingInfo(false)
    }
  }, [isRegistered])

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
        <div className="text-center text-gray-500">Загружаем опубликованную рассадку...</div>
      </div>
    )
  }

  const publishedAt = formatPublishedAt(seatingInfo?.published_at)

  return (
    <div className="min-h-screen px-4 py-4 pb-[calc(env(safe-area-inset-bottom,0px)+112px)]">
      <SectionCard>
        <SectionTitle>РАССАДКА</SectionTitle>

        {seatingInfo?.error ? (
          <div className="text-center py-4">
            <p className="mb-3 text-[19.2px] text-red-600">Не удалось загрузить опубликованную рассадку.</p>
            <p className="mb-4 text-[16.8px] text-gray-500">{seatingInfo.error}</p>
            <button
              onClick={() => void loadSeating()}
              className="rounded-lg bg-primary px-4 py-2 font-semibold text-white transition-colors hover:bg-primary-dark"
            >
              Повторить
            </button>
          </div>
        ) : !seatingInfo?.visible || seatingInfo.tables.length === 0 ? (
          <div className="text-center py-4">
            <p className="mb-2 text-[19.2px] text-gray-700">Рассадка ещё не опубликована.</p>
            <p className="text-[16.8px] text-gray-500">
              Когда организаторы нажмут «Обновить рассадку» в админ-панели, список столов появится здесь.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="rounded-lg border border-primary/15 bg-cream/30 px-4 py-3 text-center text-gray-700">
              <div className="text-[16.8px] uppercase tracking-wide text-primary/80">Опубликованная версия</div>
              <div className="mt-1 text-[18px] font-semibold text-primary-dark">
                {publishedAt ? `Обновлено ${publishedAt}` : 'Актуальная рассадка из админ-панели'}
              </div>
            </div>

            <div className="grid gap-3">
              {seatingInfo.tables.map((table, index) => (
                <SeatingTableCard key={table.table || `table-${index}`} table={table} index={index} />
              ))}
            </div>
          </div>
        )}
      </SectionCard>
    </div>
  )
}
