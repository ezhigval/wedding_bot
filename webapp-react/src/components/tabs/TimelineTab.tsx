import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { loadTimeline } from '../../utils/api'
import { useRegistration } from '../../contexts/RegistrationContext'
import type { TimelineItem } from '../../types'

export default function TimelineTab() {
  const [timeline, setTimeline] = useState<TimelineItem[]>([])
  const [loading, setLoading] = useState(true)
  const { isRegistered, isLoading: registrationLoading } = useRegistration()

  useEffect(() => {
    if (isRegistered) {
      loadTimeline().then((data) => {
        setTimeline(data)
        setLoading(false)
      })
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
    <div className="min-h-screen px-4 py-4 pb-[120px]">
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
        ) : timeline.length === 0 ? (
          <p className="text-center text-gray-500 py-4 text-[19.2px]">
            План дня будет добавлен позже
          </p>
        ) : (
          <div className="space-y-2">
            {timeline.map((item, index) => (
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
                  {item.event}
                </div>
              </motion.div>
            ))}
          </div>
        )}
      </SectionCard>
    </div>
  )
}

