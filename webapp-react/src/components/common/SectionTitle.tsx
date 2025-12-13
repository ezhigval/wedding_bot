import { motion } from 'framer-motion'

interface SectionTitleProps {
  children: React.ReactNode
  className?: string
}

export default function SectionTitle({ children, className = '' }: SectionTitleProps) {
  return (
    <motion.h2
      initial={{ opacity: 0, y: -10 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.4 }}
      className={`text-3xl md:text-4xl font-secondary font-bold text-primary text-center mb-3 tracking-wide ${className}`}
    >
      {children}
    </motion.h2>
  )
}

