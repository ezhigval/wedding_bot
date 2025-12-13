interface RankIconProps {
  rank: 'новичок' | 'любитель' | 'профи'
  className?: string
}

export default function RankIcon({ rank, className = '' }: RankIconProps) {
  const iconSize = 40
  
  if (rank === 'новичок') {
    // Комичная иконка новичка - маленький шлем или шапка
    return (
      <svg
        width={iconSize}
        height={iconSize}
        viewBox="0 0 40 40"
        className={className}
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        {/* Голова */}
        <circle cx="20" cy="22" r="10" fill="#FFE9AD" stroke="#535353" strokeWidth="2" />
        {/* Глаза */}
        <circle cx="16" cy="20" r="1.5" fill="#535353" />
        <circle cx="24" cy="20" r="1.5" fill="#535353" />
        {/* Рот - улыбка */}
        <path d="M 16 24 Q 20 27 24 24" stroke="#535353" strokeWidth="1.5" fill="none" strokeLinecap="round" />
        {/* Шлем новичка - большой и нелепый */}
        <path
          d="M 10 18 Q 20 8 30 18 L 28 12 Q 20 5 12 12 Z"
          fill="#535353"
          stroke="#3a3a3a"
          strokeWidth="1"
        />
        {/* Звездочка на шлеме */}
        <path
          d="M 20 10 L 21 13 L 24 13 L 21.5 15 L 22.5 18 L 20 16 L 17.5 18 L 18.5 15 L 16 13 L 19 13 Z"
          fill="#FFE9AD"
        />
      </svg>
    )
  }
  
  if (rank === 'любитель') {
    // Комичная иконка любителя - корона или медаль
    return (
      <svg
        width={iconSize}
        height={iconSize}
        viewBox="0 0 40 40"
        className={className}
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        {/* Голова */}
        <circle cx="20" cy="22" r="10" fill="#FFE9AD" stroke="#535353" strokeWidth="2" />
        {/* Глаза */}
        <circle cx="16" cy="20" r="1.5" fill="#535353" />
        <circle cx="24" cy="20" r="1.5" fill="#535353" />
        {/* Рот - довольная улыбка */}
        <path d="M 14 24 Q 20 28 26 24" stroke="#535353" strokeWidth="1.5" fill="none" strokeLinecap="round" />
        {/* Корона любителя */}
        <path
          d="M 12 15 L 15 10 L 18 12 L 20 8 L 22 12 L 25 10 L 28 15 L 28 18 L 12 18 Z"
          fill="#FFD700"
          stroke="#FFA500"
          strokeWidth="1.5"
        />
        {/* Звезды на короне */}
        <circle cx="15" cy="12" r="1.5" fill="#FFE9AD" />
        <circle cx="20" cy="9" r="1.5" fill="#FFE9AD" />
        <circle cx="25" cy="12" r="1.5" fill="#FFE9AD" />
      </svg>
    )
  }
  
  // Комичная иконка профи - супергеройская маска или шлем чемпиона
  return (
    <svg
      width={iconSize}
      height={iconSize}
      viewBox="0 0 40 40"
      className={className}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Голова */}
      <circle cx="20" cy="22" r="10" fill="#FFE9AD" stroke="#535353" strokeWidth="2" />
      {/* Глаза - уверенные */}
      <ellipse cx="16" cy="20" rx="2" ry="2.5" fill="#535353" />
      <ellipse cx="24" cy="20" rx="2" ry="2.5" fill="#535353" />
      {/* Рот - серьезная улыбка */}
      <path d="M 14 25 Q 20 29 26 25" stroke="#535353" strokeWidth="2" fill="none" strokeLinecap="round" />
      {/* Маска профи - стильная маска супергероя */}
      <path
        d="M 12 18 Q 12 14 16 12 Q 20 10 24 12 Q 28 14 28 18 L 28 20 Q 28 22 26 23 Q 24 24 20 24 Q 16 24 14 23 Q 12 22 12 20 Z"
        fill="#535353"
        stroke="#3a3a3a"
        strokeWidth="1.5"
      />
      {/* Молния на маске */}
      <path
        d="M 20 12 L 18 16 L 20 16 L 19 20 L 22 16 L 20 16 Z"
        fill="#FFE9AD"
        stroke="#FFD700"
        strokeWidth="0.5"
      />
      {/* Звезды вокруг */}
      <path
        d="M 8 15 L 8.5 16.5 L 10 16 L 8.5 17.5 L 9 19 L 8 18 L 7 19 L 7.5 17.5 L 6 16 L 7.5 16.5 Z"
        fill="#FFD700"
        opacity="0.8"
      />
      <path
        d="M 32 15 L 32.5 16.5 L 34 16 L 32.5 17.5 L 33 19 L 32 18 L 31 19 L 31.5 17.5 L 30 16 L 31.5 16.5 Z"
        fill="#FFD700"
        opacity="0.8"
      />
    </svg>
  )
}

