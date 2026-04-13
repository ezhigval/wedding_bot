import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'
import { useUser } from '../../contexts/UserContext'
import { getSeatingInfo, type SeatingInfo } from '../../utils/api'

export default function SeatingTab() {
  const { isRegistered, isLoading } = useRegistration()
  const { userId, manualUsername } = useUser()
  const [seatingInfo, setSeatingInfo] = useState<SeatingInfo | null>(null)
  const [loadingInfo, setLoadingInfo] = useState(true)

  useEffect(() => {
    if (!isRegistered) {
      setLoadingInfo(false)
      return
    }

    let isCancelled = false

    const loadSeating = async () => {
      setLoadingInfo(true)
      const result = await getSeatingInfo({ userId, username: manualUsername })
      if (!isCancelled) {
        setSeatingInfo(result)
        setLoadingInfo(false)
      }
    }

    loadSeating()

    return () => {
      isCancelled = true
    }
  }, [isRegistered, userId, manualUsername])

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
        <div className="text-center text-gray-500">Загружаем вашу рассадку...</div>
      </div>
    )
  }

  const neighbors = seatingInfo?.neighbors || []

  return (
    <div className="min-h-screen px-4 py-4 pb-[calc(env(safe-area-inset-bottom,0px)+112px)]">
      <SectionCard>
        <SectionTitle>РАССАДКА</SectionTitle>

        {seatingInfo?.error ? (
          <div className="text-center py-4">
            <p className="text-[19.2px] text-red-600 mb-2">Не удалось загрузить данные рассадки.</p>
            <p className="text-[16.8px] text-gray-500">{seatingInfo.error}</p>
          </div>
        ) : !seatingInfo?.visible ? (
          <div className="text-center py-4">
            <p className="text-[19.2px] text-gray-700 mb-2">Ваш стол пока не назначен.</p>
            <p className="text-[16.8px] text-gray-500">
              Как только рассадка будет зафиксирована, здесь появится ваш стол и соседи.
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {seatingInfo.full_name && (
              <motion.div
                initial={{ opacity: 0, y: 12 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.3 }}
                className="rounded-lg border border-primary/20 bg-cream/40 px-4 py-3 text-center"
              >
                <div className="text-[16.8px] uppercase tracking-wide text-primary/80">Гость</div>
                <div className="text-[24px] font-semibold text-primary-dark">{seatingInfo.full_name}</div>
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
                {seatingInfo.table || 'Без номера'}
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
                  Список соседей появится после финальной фиксации рассадки.
                </p>
              ) : (
                <div className="grid gap-2">
                  {neighbors.map((neighbor) => (
                    <div
                      key={neighbor}
                      className="rounded-lg bg-cream/40 px-3 py-2 text-center text-[18px] text-gray-700"
                    >
                      {neighbor}
                    </div>
                  ))}
                </div>
              )}
            </motion.div>
          </div>
        )}
      </SectionCard>
    </div>
  )
}
