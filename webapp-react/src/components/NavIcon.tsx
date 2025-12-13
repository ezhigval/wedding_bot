interface NavIconProps {
  name: 'home' | 'dresscode' | 'timeline' | 'seating' | 'menu'
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
  }

  return icons[name]
}

