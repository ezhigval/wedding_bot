interface NavIconProps {
  name: 'home' | 'challenge' | 'menu' | 'photo' | 'timeline' | 'dresscode' | 'seating' | 'wishes'
  isActive: boolean
  className?: string
}

export default function NavIcon({ name, isActive, className = '' }: NavIconProps) {
  const fillColor = isActive ? '#5A7C52' : '#DBD0C4'
  const strokeColor = isActive ? '#4A6B42' : '#999999'
  const strokeWidth = 1.5

  const icons = {
    home: (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
      >
        {/* Крыша */}
        <path
          d="M3 12L12 3L21 12"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        {/* Стены дома */}
        <rect
          x="6"
          y="12"
          width="12"
          height="9"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Дверь */}
        <rect
          x="10"
          y="16"
          width="4"
          height="5"
          fill={isActive ? '#FFE9AD' : '#FFFFFF'}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
      </svg>
    ),
    dresscode: (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
      >
        {/* Воротник */}
        <path
          d="M8 8L12 4L16 8"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        {/* Платье/рубашка */}
        <path
          d="M8 8V20C8 20.5 8.5 21 9 21H15C15.5 21 16 20.5 16 20V8"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Рукава */}
        <path
          d="M6 10L8 8L6 12"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <path
          d="M18 10L16 8L18 12"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    ),
    timeline: (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
      >
        {/* Циферблат */}
        <circle
          cx="12"
          cy="12"
          r="8"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Центр */}
        <circle
          cx="12"
          cy="12"
          r="1.5"
          fill={strokeColor}
        />
        {/* Стрелки */}
        <line
          x1="12"
          y1="12"
          x2="12"
          y2="7"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
        <line
          x1="12"
          y1="12"
          x2="16"
          y2="12"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
      </svg>
    ),
    seating: (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
      >
        {/* Спинка стула */}
        <rect
          x="6"
          y="4"
          width="12"
          height="3"
          rx="1"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Сиденье */}
        <rect
          x="7"
          y="10"
          width="10"
          height="3"
          rx="1"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Ножки */}
        <line
          x1="8"
          y1="13"
          x2="8"
          y2="20"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
        <line
          x1="16"
          y1="13"
          x2="16"
          y2="20"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
        {/* Перекладина */}
        <line
          x1="8"
          y1="17"
          x2="16"
          y2="17"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
      </svg>
    ),
    menu: (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
      >
        {/* Линии меню */}
        <rect
          x="4"
          y="6"
          width="16"
          height="2"
          rx="1"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        <rect
          x="4"
          y="11"
          width="16"
          height="2"
          rx="1"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        <rect
          x="4"
          y="16"
          width="16"
          height="2"
          rx="1"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Иконка документа справа */}
        <rect
          x="16"
          y="4"
          width="4"
          height="6"
          rx="0.5"
          fill={isActive ? '#FFE9AD' : '#FFFFFF'}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        <line
          x1="17"
          y1="6"
          x2="19"
          y2="6"
          stroke={strokeColor}
          strokeWidth={0.8}
          strokeLinecap="round"
        />
        <line
          x1="17"
          y1="8"
          x2="19"
          y2="8"
          stroke={strokeColor}
          strokeWidth={0.8}
          strokeLinecap="round"
        />
      </svg>
    ),
    wishes: (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
      >
        {/* Сердце */}
        <path
          d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Звездочка внутри */}
        <path
          d="M12 8L13.09 10.26L15.5 10.61L13.68 12.39L14.18 14.79L12 13.5L9.82 14.79L10.32 12.39L8.5 10.61L10.91 10.26L12 8Z"
          fill={isActive ? '#FFE9AD' : '#FFFFFF'}
          stroke={strokeColor}
          strokeWidth={0.5}
        />
      </svg>
    ),
    challenge: (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
      >
        {/* Щит/медаль */}
        <path
          d="M12 2L4 5V11C4 16.55 7.16 21.74 12 23C16.84 21.74 20 16.55 20 11V5L12 2Z"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Звезда внутри */}
        <path
          d="M12 7L13.09 9.26L15.5 9.61L13.68 11.39L14.18 13.79L12 12.5L9.82 13.79L10.32 11.39L8.5 9.61L10.91 9.26L12 7Z"
          fill={isActive ? '#FFE9AD' : '#FFFFFF'}
          stroke={strokeColor}
          strokeWidth={0.5}
        />
      </svg>
    ),
    photo: (
      <svg
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
      >
        {/* Камера */}
        <rect
          x="4"
          y="6"
          width="16"
          height="12"
          rx="2"
          fill={fillColor}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Объектив */}
        <circle
          cx="12"
          cy="12"
          r="3"
          fill={isActive ? '#FFE9AD' : '#FFFFFF'}
          stroke={strokeColor}
          strokeWidth={strokeWidth}
        />
        {/* Вспышка */}
        <rect
          x="16"
          y="8"
          width="2"
          height="2"
          rx="0.5"
          fill={strokeColor}
        />
        {/* Ремешок */}
        <path
          d="M6 8L5 6M18 8L19 6"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
      </svg>
    ),
  }

  return icons[name]
}

