import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { loadTimeline } from '../../utils/api'
import { useRegistration } from '../../contexts/RegistrationContext'
import type { TimelineItem } from '../../types'

function splitEventAndAddress(event: string): { title: string; address?: string } {
  const trimmed = event.trim()
  // Поддерживаем формат: "Событие (адрес)" — адрес берём только из завершающих скобок,
  // чтобы не ломать возможные скобки внутри текста.
  const match = trimmed.match(/^(.*?)(?:\s*\(([^()]+)\))\s*$/)
  if (!match) {
    return { title: trimmed }
  }

  const title = match[1].trim()
  const address = match[2].trim()
  if (!title || !address) {
    return { title: trimmed }
  }

  return { title, address }
}

export default function TimelineTab() {
  const [timeline, setTimeline] = useState<TimelineItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const { isRegistered, isLoading: registrationLoading } = useRegistration()

  const loadTimelineData = async () => {
    setLoading(true)
    const result = await loadTimeline()
    setTimeline(result.timeline)
    setError(result.error || null)
    setLoading(false)
  }

  useEffect(() => {
    if (isRegistered) {
      void loadTimelineData()
    } else {
      setTimeline([])
      setError(null)
      setLoading(false)
    }
  }, [isRegistered])

  if (registrationLoading) {
    return (
      <div className="min-h-screen px-4 py-4 flex items-center justify-center">
        <div className="text-center text-gray-500">Загрузка...</div>
      </div>
    )
  }

  if (!isRegistered) {
    return <RegistrationRequired />
  }

  return (
    <div className="min-h-screen px-4 py-4 pb-[calc(env(safe-area-inset-bottom,0px)+112px)]">
      <SectionCard>
        <SectionTitle>ПЛАН ДНЯ</SectionTitle>
        {loading ? (
          <div className="text-center py-4">
            <motion.div
              animate={{ rotate: 360 }}
              transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
              className="w-8 h-8 border-4 border-primary border-t-transparent rounded-full mx-auto"
            />
          </div>
        ) : error ? (
          <div className="text-center py-4">
            <p className="text-[19.2px] text-red-600 mb-3">{error}</p>
            <button
              onClick={() => void loadTimelineData()}
              className="rounded-lg bg-primary px-4 py-2 font-semibold text-white transition-colors hover:bg-primary-dark"
            >
              Повторить
            </button>
          </div>
        ) : timeline.length === 0 ? (
          <div className="text-center py-4">
            <p className="text-[19.2px] text-gray-700 mb-2">План дня пока не опубликован.</p>
            <p className="text-[16.8px] text-gray-500">
              Когда организаторы зафиксируют расписание, оно появится здесь автоматически.
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {timeline.map((item, index) => (
              (() => {
                const { title, address } = splitEventAndAddress(item.event)
                return (
              <motion.div
                key={index}
                initial={{ opacity: 0, x: -20 }}
                whileInView={{ opacity: 1, x: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.4, delay: index * 0.1 }}
                className="flex flex-col md:flex-row gap-2 p-[3.2px] bg-cream/30 rounded-lg border-l-4 border-primary"
              >
                <div className="font-semibold text-primary min-w-[120px] md:text-right leading-[1.2]" style={{ fontSize: '23.4px' }}>
                  {item.time}
                </div>
                <div className="flex-1 text-gray-700 text-[19.2px] md:text-[21.6px] leading-[1.2]">
                  <div>{title}</div>
                  {address && (
                    <div className="mt-1 text-gray-500 text-[16.8px] md:text-[19.2px] leading-[1.2]">
                      <span className="font-semibold text-gray-600">Адрес:</span> {address}
                    </div>
                  )}
                </div>
              </motion.div>
                )
              })()
            ))}
          </div>
        )}
      </SectionCard>
    </div>
  )
}
