# Wedding Mini App Frontend

Исходники Telegram Mini App. Production bundle собирается в `../webapp/` и затем раздаётся Go-сервисом.

## Команды

```bash
npm ci
npm run lint
npm run build
npm run dev
```

По умолчанию Vite proxy направляет `/api` на `http://localhost:10000`.

## Важное

- `webapp-react/.vite/` не должен попадать в Git.
- Production runtime использует только `webapp/`, а не исходники из `webapp-react/`.
- Если меняется контракт API, обновляйте корневую документацию проекта.
