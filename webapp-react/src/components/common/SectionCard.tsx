import { motion } from 'framer-motion'
import { ReactNode } from 'react'

interface SectionCardProps {
  children: ReactNode
  className?: string
}

export default function SectionCard({ children, className = '' }: SectionCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: '-50px' }}
      transition={{ duration: 0.5 }}
      className={`bg-white/90 backdrop-blur-sm rounded-lg p-4 md:p-[6.4px] mb-2.5 shadow-md ${className}`}
    >
      {children}
    </motion.div>
  )
}

