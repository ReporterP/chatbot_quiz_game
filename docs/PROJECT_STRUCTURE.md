# Quiz Game — Структура проекта

```
chatbot_quiz_game/
├── docs/                          # Документация проекта
│   ├── PROJECT_OVERVIEW.md        # Обзор, стек, архитектура
│   ├── USER_FLOWS.md              # Пользовательские сценарии
│   ├── API_DESIGN.md              # Проектирование API
│   ├── DATABASE_SCHEMA.md         # Схема БД
│   └── PROJECT_STRUCTURE.md       # Этот файл
│
├── backend/                       # Go бэкенд (Gin + GORM)
│   ├── cmd/
│   │   └── server/
│   │       └── main.go            # Точка входа
│   ├── internal/
│   │   ├── config/                # Конфигурация (env)
│   │   │   └── config.go
│   │   ├── models/                # GORM-модели
│   │   │   ├── host.go
│   │   │   ├── quiz.go
│   │   │   ├── question.go
│   │   │   ├── option.go
│   │   │   ├── session.go
│   │   │   ├── participant.go
│   │   │   └── answer.go
│   │   ├── handlers/              # HTTP-обработчики (Gin)
│   │   │   ├── auth.go
│   │   │   ├── quiz.go
│   │   │   ├── question.go
│   │   │   ├── session.go
│   │   │   └── participant.go
│   │   ├── middleware/            # Middleware (JWT, CORS, bot auth)
│   │   │   └── auth.go
│   │   ├── services/             # Бизнес-логика
│   │   │   ├── auth.go
│   │   │   ├── quiz.go
│   │   │   ├── session.go
│   │   │   └── scoring.go
│   │   ├── ws/                   # WebSocket для real-time
│   │   │   └── hub.go
│   │   └── database/             # Подключение к БД
│   │       └── database.go
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── frontend/                      # React SPA
│   ├── public/
│   ├── src/
│   │   ├── api/                   # HTTP-клиент к API
│   │   ├── store/                 # Redux store + slices
│   │   ├── pages/                 # Страницы (Router)
│   │   │   ├── LoginPage.jsx
│   │   │   ├── RegisterPage.jsx
│   │   │   ├── DashboardPage.jsx  # Список квизов
│   │   │   ├── QuizEditPage.jsx   # Редактор квиза
│   │   │   ├── SessionLobbyPage.jsx   # QR-код, ожидание
│   │   │   ├── SessionGamePage.jsx    # Экран ведущего (вопросы)
│   │   │   └── LeaderboardPage.jsx    # Таблица лидеров
│   │   ├── components/            # Переиспользуемые компоненты
│   │   ├── hooks/                 # Кастомные хуки
│   │   ├── App.jsx
│   │   └── index.jsx
│   ├── package.json
│   └── Dockerfile
│
├── bot/                           # Aiogram Telegram-бот
│   ├── bot/
│   │   ├── __init__.py
│   │   ├── main.py                # Точка входа
│   │   ├── config.py              # Конфигурация
│   │   ├── api_client.py          # HTTP-клиент к Go API
│   │   ├── handlers/              # Обработчики команд/callback
│   │   │   ├── __init__.py
│   │   │   ├── start.py           # /start, ввод кода сессии
│   │   │   ├── join.py            # Ввод ника, присоединение
│   │   │   └── game.py            # Ответы на вопросы
│   │   ├── keyboards/             # Inline-клавиатуры
│   │   │   └── __init__.py
│   │   └── states/                # FSM-состояния
│   │       └── __init__.py
│   ├── requirements.txt
│   └── Dockerfile
│
├── docker-compose.yml             # Оркестрация всех сервисов
├── .env.example                   # Пример переменных окружения
├── .gitignore
└── README.md
```

---

## Сервисы в Docker Compose

| Сервис     | Порт  | Описание                          |
|------------|-------|-----------------------------------|
| `db`       | 5432  | PostgreSQL                        |
| `backend`  | 8080  | Go API (Gin)                      |
| `frontend` | 3000  | React SPA (dev) / nginx (prod)    |
| `bot`      | —     | Telegram-бот (без публичного порта)|

## Связи между сервисами

- `backend` → `db` (подключение к PostgreSQL)
- `frontend` → `backend` (HTTP API запросы)
- `bot` → `backend` (HTTP API запросы)
