import { motion } from 'framer-motion'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'

export default function WishesTab() {
  const { isRegistered, isLoading } = useRegistration()
  const wishes = [
    'Просим не дарить нам цветы, прекрасной альтернативой станет бутылка хорошего вина для нашей семейной винотеки.',
    'Если хотите сделать нам ценный и нужный подарок, мы будем признательны за вклад в бюджет нашей молодой семьи.',
  ]

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
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5 }}
        className="bg-primary-dark rounded-lg p-4 md:p-[6.4px] mb-2.5 shadow-md"
      >
        <motion.h2
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
          className="text-3xl md:text-4xl font-secondary font-bold text-white mb-3 text-center leading-[1.2]"
        >
          ПОЖЕЛАНИЯ
        </motion.h2>

        <div className="mt-5 space-y-[54px]">
          {wishes.map((wish, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              className="flex items-start gap-4 px-5"
            >
              <span className="w-3 h-3 rounded-full bg-white flex-shrink-0 mt-2" />
              <p className="text-[21.6px] md:text-[21.6px] leading-[1.2] text-white text-left flex-1">
                {wish}
              </p>
            </motion.div>
          ))}
        </div>
      </motion.div>
    </div>
  )
}

