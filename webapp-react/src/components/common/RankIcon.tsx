interface RankIconProps {
  rank: 'Незнакомец' | 'Ты хто?' | 'Люся' | 'Бедный родственник' | 'Братуха' | 'Батя в здании' | 'Монстр'
  className?: string
}

export default function RankIcon({ rank, className = '' }: RankIconProps) {
  const iconSize = 40
  
  if (rank === 'Незнакомец') {
    // Незнакомец - загадочная маска с вопросительными знаками вокруг
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
        <circle cx="20" cy="22" r="10" fill="#E0E0E0" stroke="#999" strokeWidth="2" />
        {/* Глаза - закрыты, загадочные */}
        <line x1="15" y1="20" x2="19" y2="20" stroke="#666" strokeWidth="2" strokeLinecap="round" />
        <line x1="21" y1="20" x2="25" y2="20" stroke="#666" strokeWidth="2" strokeLinecap="round" />
        {/* Рот - нейтральный */}
        <line x1="16" y1="24" x2="24" y2="24" stroke="#666" strokeWidth="1.5" strokeLinecap="round" />
        {/* Маска с загадочными символами */}
        <path
          d="M 10 18 Q 20 8 30 18 L 30 15 Q 20 5 10 15 Z"
          fill="#999"
          stroke="#666"
          strokeWidth="1.5"
        />
        {/* Вопросительные знаки вокруг (загадка) */}
        <path
          d="M 6 12 Q 6 10 8 10 Q 10 10 10 12 Q 10 14 8 14 Q 6 14 6 12"
          stroke="#FF6B6B"
          strokeWidth="1.5"
          fill="none"
        />
        <circle cx="8" cy="16" r="0.8" fill="#FF6B6B" />
        <path
          d="M 34 12 Q 34 10 32 10 Q 30 10 30 12 Q 30 14 32 14 Q 34 14 34 12"
          stroke="#FF6B6B"
          strokeWidth="1.5"
          fill="none"
        />
        <circle cx="32" cy="16" r="0.8" fill="#FF6B6B" />
      </svg>
    )
  }
  
  if (rank === 'Ты хто?') {
    // Ты хто? - смешной вопросительный знак с глазами и удивленным лицом
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
        {/* Глаза - удивленные, круглые, большие */}
        <circle cx="16" cy="20" r="3.5" fill="#535353" />
        <circle cx="24" cy="20" r="3.5" fill="#535353" />
        <circle cx="16" cy="19.5" r="1.5" fill="#FFF" />
        <circle cx="24" cy="19.5" r="1.5" fill="#FFF" />
        {/* Рот - овальный, удивленный */}
        <ellipse cx="20" cy="26" rx="3.5" ry="4.5" fill="#535353" />
        {/* Большой смешной вопросительный знак с глазами над головой */}
        <path
          d="M 20 4 Q 18 2 16 4 Q 14 6 16 8 Q 18 10 20 8 Q 22 10 24 8 Q 26 6 24 4 Q 22 2 20 4"
          stroke="#FF6B6B"
          strokeWidth="3"
          fill="#FFE9AD"
          strokeLinecap="round"
        />
        {/* Глаза на вопросительном знаке */}
        <circle cx="18" cy="6" r="1.2" fill="#FF6B6B" />
        <circle cx="22" cy="6" r="1.2" fill="#FF6B6B" />
        {/* Точка внизу */}
        <circle cx="20" cy="13" r="2" fill="#FF6B6B" />
      </svg>
    )
  }
  
  if (rank === 'Люся') {
    // Люся - милое лицо с фирменным жестом Николай Петровича (оттопыренный мизинец)
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
        {/* Глаза - милые, с бликами */}
        <circle cx="16" cy="20" r="2" fill="#535353" />
        <circle cx="24" cy="20" r="2" fill="#535353" />
        <circle cx="16.5" cy="19.5" r="0.8" fill="#FFF" />
        <circle cx="24.5" cy="19.5" r="0.8" fill="#FFF" />
        {/* Рот - улыбка */}
        <path d="M 16 24 Q 20 27 24 24" stroke="#535353" strokeWidth="1.5" fill="none" strokeLinecap="round" />
        {/* Бантик на голове */}
        <path
          d="M 15 12 L 17 8 L 20 10 L 23 8 L 25 12 L 20 14 Z"
          fill="#FF69B4"
          stroke="#FF1493"
          strokeWidth="1"
        />
        <circle cx="18" cy="10" r="1" fill="#FFF" />
        <circle cx="22" cy="10" r="1" fill="#FFF" />
        {/* Фирменный жест - рука с оттопыренным мизинцем (справа) */}
        <path
          d="M 30 25 L 32 23 L 34 25 L 34 30 L 32 32 L 30 30 Z"
          fill="#FFE9AD"
          stroke="#535353"
          strokeWidth="1.5"
        />
        {/* Пальцы - все сжаты, кроме мизинца */}
        <line x1="31" y1="26" x2="31" y2="29" stroke="#535353" strokeWidth="1.5" strokeLinecap="round" />
        <line x1="32" y1="26" x2="32" y2="29" stroke="#535353" strokeWidth="1.5" strokeLinecap="round" />
        <line x1="33" y1="26" x2="33" y2="29" stroke="#535353" strokeWidth="1.5" strokeLinecap="round" />
        {/* Мизинец - оттопыренный! */}
        <path
          d="M 33.5 29 Q 35 28 36 30 Q 35 32 33.5 31"
          stroke="#FF6B6B"
          strokeWidth="2.5"
          fill="none"
          strokeLinecap="round"
        />
        <circle cx="35.5" cy="30" r="1" fill="#FF6B6B" />
      </svg>
    )
  }
  
  if (rank === 'Бедный родственник') {
    // Бедный родственник - грустное лицо, дырявая шапка, слезы, заплатки
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
        {/* Глаза - грустные, опущенные вниз */}
        <path d="M 14 20 Q 16 18 18 20" stroke="#535353" strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 22 20 Q 24 18 26 20" stroke="#535353" strokeWidth="2.5" fill="none" strokeLinecap="round" />
        {/* Слезы - капают */}
        <path d="M 15 22 Q 15 25 15 28" stroke="#87CEEB" strokeWidth="2" fill="none" strokeLinecap="round" />
        <circle cx="15" cy="28" r="1.5" fill="#87CEEB" />
        <path d="M 25 22 Q 25 25 25 28" stroke="#87CEEB" strokeWidth="2" fill="none" strokeLinecap="round" />
        <circle cx="25" cy="28" r="1.5" fill="#87CEEB" />
        {/* Рот - грустный, опущенный вниз */}
        <path d="M 16 26 Q 20 23 24 26" stroke="#535353" strokeWidth="2.5" fill="none" strokeLinecap="round" />
        {/* Дырявая шапка с заплатками */}
        <rect x="10" y="12" width="20" height="8" rx="2" fill="#8B4513" stroke="#654321" strokeWidth="1" />
        <ellipse cx="20" cy="12" rx="10" ry="3" fill="#8B4513" />
        {/* Дыра в шапке */}
        <circle cx="18" cy="14" r="2" fill="#FFE9AD" />
        {/* Заплатки */}
        <rect x="15" y="13" width="4" height="3" rx="0.5" fill="#654321" opacity="0.7" />
        <line x1="16" y1="13" x2="18" y2="16" stroke="#8B4513" strokeWidth="0.5" />
        <line x1="18" y1="13" x2="19" y2="16" stroke="#8B4513" strokeWidth="0.5" />
        <rect x="22" y="14" width="3" height="2" rx="0.5" fill="#654321" opacity="0.7" />
      </svg>
    )
  }
  
  if (rank === 'Братуха') {
    // Братуха - уверенное лицо, стильная кепка, солнечные очки, звездочка
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
        {/* Солнечные очки */}
        <rect x="12" y="18" width="8" height="4" rx="2" fill="#1a1a1a" opacity="0.8" />
        <rect x="20" y="18" width="8" height="4" rx="2" fill="#1a1a1a" opacity="0.8" />
        <line x1="20" y1="19" x2="20" y2="21" stroke="#FFD700" strokeWidth="0.5" />
        {/* Рот - уверенная улыбка */}
        <path d="M 14 25 Q 20 28 26 25" stroke="#535353" strokeWidth="2" fill="none" strokeLinecap="round" />
        {/* Стильная кепка */}
        <path
          d="M 10 18 Q 20 10 30 18 L 28 14 Q 20 8 12 14 Z"
          fill="#1E90FF"
          stroke="#0000CD"
          strokeWidth="1.5"
        />
        <ellipse cx="20" cy="16" rx="10" ry="4" fill="#1E90FF" />
        {/* Козырек кепки */}
        <ellipse cx="20" cy="18" rx="8" ry="2" fill="#000" opacity="0.8" />
        {/* Звездочка на кепке */}
        <path
          d="M 20 12 L 20.8 14 L 23 14 L 21.1 15.5 L 21.8 17.5 L 20 16.5 L 18.2 17.5 L 18.9 15.5 L 17 14 L 19.2 14 Z"
          fill="#FFD700"
        />
      </svg>
    )
  }
  
  if (rank === 'Батя в здании') {
    // Батя в здании - серьезное лицо, роскошная корона, борода, усы, властный вид
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
        {/* Глаза - серьезные, пронзительные */}
        <ellipse cx="16" cy="20" rx="2.5" ry="3.5" fill="#535353" />
        <ellipse cx="24" cy="20" rx="2.5" ry="3.5" fill="#535353" />
        <circle cx="16" cy="19.5" r="0.8" fill="#FFF" />
        <circle cx="24" cy="19.5" r="0.8" fill="#FFF" />
        {/* Брови - серьезные, нахмуренные */}
        <path d="M 13 17 Q 16 16 18 17" stroke="#535353" strokeWidth="2.5" fill="none" strokeLinecap="round" />
        <path d="M 22 17 Q 24 16 27 17" stroke="#535353" strokeWidth="2.5" fill="none" strokeLinecap="round" />
        {/* Рот - серьезная улыбка */}
        <path d="M 14 26 Q 20 29 26 26" stroke="#535353" strokeWidth="2" fill="none" strokeLinecap="round" />
        {/* Усы - пышные */}
        <path d="M 16 25 Q 18 24 20 25" stroke="#8B4513" strokeWidth="2" fill="none" strokeLinecap="round" />
        <path d="M 20 25 Q 22 24 24 25" stroke="#8B4513" strokeWidth="2" fill="none" strokeLinecap="round" />
        {/* Роскошная корона */}
        <path
          d="M 12 15 L 15 10 L 18 12 L 20 8 L 22 12 L 25 10 L 28 15 L 28 18 L 12 18 Z"
          fill="#FFD700"
          stroke="#FFA500"
          strokeWidth="1.5"
        />
        {/* Драгоценности на короне */}
        <circle cx="15" cy="12" r="1.8" fill="#FF1493" />
        <circle cx="15" cy="12" r="0.8" fill="#FFF" />
        <circle cx="20" cy="9" r="2.5" fill="#00CED1" />
        <circle cx="20" cy="9" r="1" fill="#FFF" />
        <circle cx="25" cy="12" r="1.8" fill="#FF1493" />
        <circle cx="25" cy="12" r="0.8" fill="#FFF" />
        {/* Пышная борода */}
        <path
          d="M 14 28 Q 20 36 26 28 Q 20 38 14 28"
          fill="#8B4513"
          stroke="#654321"
          strokeWidth="1.5"
        />
        {/* Разделение бороды */}
        <line x1="20" y1="28" x2="20" y2="36" stroke="#654321" strokeWidth="0.8" opacity="0.6" />
      </svg>
    )
  }
  
  // Монстр - самое крутое звание, устрашающее лицо, рога, огонь, злость
  return (
    <svg
      width={iconSize}
      height={iconSize}
      viewBox="0 0 40 40"
      className={className}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Голова - красная, злая */}
      <circle cx="20" cy="22" r="10" fill="#FF6B6B" stroke="#8B0000" strokeWidth="2.5" />
      {/* Глаза - злые, красные, горящие */}
      <ellipse cx="16" cy="20" rx="3.5" ry="4" fill="#FF0000" />
      <ellipse cx="24" cy="20" rx="3.5" ry="4" fill="#FF0000" />
      <circle cx="16" cy="19.5" r="1.2" fill="#FFF" />
      <circle cx="24" cy="19.5" r="1.2" fill="#FFF" />
      {/* Брови - злые, нахмуренные */}
      <path d="M 12 17 Q 16 15 18 17" stroke="#8B0000" strokeWidth="3" fill="none" strokeLinecap="round" />
      <path d="M 22 17 Q 24 15 28 17" stroke="#8B0000" strokeWidth="3" fill="none" strokeLinecap="round" />
      {/* Рот - злой, с зубами, открытый */}
      <path d="M 14 26 Q 20 30 26 26" stroke="#8B0000" strokeWidth="2.5" fill="none" strokeLinecap="round" />
      {/* Зубы - острые */}
      <rect x="17" y="25" width="1.5" height="4" fill="#FFF" rx="0.5" />
      <rect x="20" y="25" width="1.5" height="4" fill="#FFF" rx="0.5" />
      <rect x="23" y="25" width="1.5" height="4" fill="#FFF" rx="0.5" />
      {/* Рога - большие, устрашающие */}
      <path
        d="M 12 18 Q 10 10 6 8 Q 8 6 10 8 Q 12 10 12 18"
        fill="#8B0000"
        stroke="#000"
        strokeWidth="1.5"
      />
      <path
        d="M 28 18 Q 30 10 34 8 Q 32 6 30 8 Q 28 10 28 18"
        fill="#8B0000"
        stroke="#000"
        strokeWidth="1.5"
      />
      {/* Огонь вокруг головы */}
      <path
        d="M 8 15 Q 10 12 12 15 Q 10 18 8 15"
        fill="#FF4500"
        opacity="0.9"
      />
      <path
        d="M 32 15 Q 30 12 28 15 Q 30 18 32 15"
        fill="#FF4500"
        opacity="0.9"
      />
      <path
        d="M 5 22 Q 7 20 9 22 Q 7 24 5 22"
        fill="#FF6347"
        opacity="0.8"
      />
      <path
        d="M 35 22 Q 33 20 31 22 Q 33 24 35 22"
        fill="#FF6347"
        opacity="0.8"
      />
      {/* Звезды вокруг (сила, мощь) */}
      <path
        d="M 5 20 L 5.5 21.5 L 7 21 L 5.5 22.5 L 6 24 L 5 23 L 4 24 L 4.5 22.5 L 3 21 L 4.5 21.5 Z"
        fill="#FFD700"
        opacity="1"
      />
      <path
        d="M 35 20 L 35.5 21.5 L 37 21 L 35.5 22.5 L 36 24 L 35 23 L 34 24 L 34.5 22.5 L 33 21 L 34.5 21.5 Z"
        fill="#FFD700"
        opacity="1"
      />
      {/* Шрамы */}
      <line x1="18" y1="22" x2="22" y2="24" stroke="#8B0000" strokeWidth="1" opacity="0.7" />
    </svg>
  )
}
