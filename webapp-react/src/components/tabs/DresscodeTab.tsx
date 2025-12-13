import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'

export default function DresscodeTab() {
  const { isRegistered, isLoading } = useRegistration()

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

  return (
    <div className="min-h-screen px-4 py-4">
      <SectionCard>
        <SectionTitle>ДРЕСС-КОД</SectionTitle>
        <motion.p
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="text-[19.2px] md:text-[21.6px] leading-[1.2] text-gray-700 text-center"
        >
          Мы очень старались сделать праздник красивым и будем рады, если вы поддержите цветовую
          палитру торжества.
        </motion.p>
      </SectionCard>

      <SectionCard>
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          whileInView={{ opacity: 1, scale: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="w-full"
        >
          <img
            src="/res/dresscode/dresscode_animation_final.png"
            alt="Дресс-код"
            className="w-full h-auto rounded-lg"
            onError={(e) => {
              ;(e.target as HTMLImageElement).style.display = 'none'
            }}
          />
        </motion.div>
      </SectionCard>
    </div>
  )
}

