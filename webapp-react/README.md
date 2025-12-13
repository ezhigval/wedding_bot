# Wedding Mini App - React Version

Современная версия Telegram Mini App для свадебного бота, построенная на React + TypeScript + Vite.

## Технологический стек

- **React 18** - UI библиотека
- **TypeScript** - типизация
- **Vite** - сборщик и dev-сервер
- **Framer Motion** - плавные анимации
- **Tailwind CSS** - стилизация
- **React Router** - навигация (опционально)

## Установка

```bash
cd webapp-react
npm install
```

## Разработка

```bash
npm run dev
```

Приложение будет доступно на `http://localhost:5173`

Vite автоматически проксирует запросы к `/api` на `http://localhost:10000` (Python сервер)

## Сборка

```bash
npm run build
```

Собранные файлы будут в папке `../webapp/` (родительская директория)

## Структура проекта

```
webapp-react/
├── src/
│   ├── components/       # React компоненты
│   │   ├── tabs/         # Компоненты вкладок
│   │   └── common/       # Общие компоненты
│   ├── hooks/            # Custom hooks
│   ├── utils/            # Утилиты (API, Telegram)
│   ├── types/            # TypeScript типы
│   ├── App.tsx           # Главный компонент
│   └── main.tsx          # Точка входа
├── public/               # Статические файлы
├── index.html            # HTML шаблон
└── vite.config.ts        # Конфигурация Vite
```

## Интеграция с Python сервером

После сборки файлы попадают в `webapp/`, откуда Python сервер (`server.py`) раздает их как статические файлы.

## Особенности

- ✅ Плавные анимации с Framer Motion
- ✅ TypeScript для надежности
- ✅ Компонентная архитектура
- ✅ Быстрая разработка с HMR
- ✅ Оптимизированная сборка

