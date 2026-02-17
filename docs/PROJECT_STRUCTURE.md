# Quiz Game — Структура проекта

```
chatbot_quiz_game/
├── .github/
│   └── workflows/
│       └── deploy.yml                 # CI/CD — GitHub Actions (SSH deploy на VPS)
│
├── backend/                           # Go бэкенд (Gin + GORM + Telegram Bot Manager)
│   ├── cmd/
│   │   └── server/
│   │       └── main.go                # Точка входа: роутинг, миграции, запуск BotManager
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go              # Загрузка конфигурации из .env
│   │   ├── database/
│   │   │   └── database.go            # Подключение к PostgreSQL
│   │   ├── models/                    # GORM-модели
│   │   │   ├── host.go
│   │   │   ├── quiz.go
│   │   │   ├── category.go
│   │   │   ├── question.go
│   │   │   ├── question_image.go
│   │   │   ├── option.go
│   │   │   ├── session.go
│   │   │   ├── participant.go
│   │   │   ├── answer.go
│   │   │   └── telegram_user.go
│   │   ├── handlers/                  # HTTP-обработчики (Gin)
│   │   │   ├── auth.go                # Регистрация / логин
│   │   │   ├── quiz.go                # CRUD квизов, категорий, вопросов, reorder
│   │   │   ├── quiz_export.go         # Экспорт/импорт квизов (JSON, CSV)
│   │   │   ├── question.go            # Обновление/удаление вопросов, загрузка картинок
│   │   │   ├── session.go             # Сессии: создание, управление, завершение
│   │   │   ├── participant.go         # Присоединение участников, ответы
│   │   │   ├── settings.go            # Настройки ведущего (bot_token, bot_link)
│   │   │   ├── telegram_user.go       # Telegram-пользователи: ник, история
│   │   │   ├── ws.go                  # WebSocket endpoint
│   │   │   └── common.go              # Общие хелперы
│   │   ├── middleware/
│   │   │   └── auth.go                # JWT middleware, Bot API Key middleware
│   │   ├── services/                  # Бизнес-логика
│   │   │   ├── auth.go                # Регистрация, логин, JWT
│   │   │   ├── quiz.go                # CRUD квизов, импорт вопросов
│   │   │   ├── session.go             # Управление сессиями, подсчёт очков
│   │   │   ├── scoring.go             # Расчёт очков (+100 за правильный, speed bonus)
│   │   │   └── telegram_user.go       # CRUD Telegram-пользователей
│   │   ├── telegram/                  # Telegram Bot Manager (Go, webhooks)
│   │   │   ├── types.go               # Типы Telegram API (Update, Message, etc.)
│   │   │   ├── client.go              # HTTP-клиент к Telegram Bot API
│   │   │   ├── state.go               # In-memory FSM (состояния пользователей)
│   │   │   ├── keyboards.go           # Конструкторы inline/reply клавиатур
│   │   │   ├── handler.go             # Обработка входящих Update (сообщения, callback)
│   │   │   ├── tracker.go             # Отслеживание сессий, отправка обновлений участникам
│   │   │   └── manager.go             # BotManager: динамическое управление ботами
│   │   └── ws/
│   │       └── hub.go                 # WebSocket hub (broadcast обновлений сессии)
│   ├── docs/                          # Swagger (auto-generated)
│   │   ├── docs.go
│   │   ├── swagger.json
│   │   └── swagger.yaml
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── frontend/                          # React SPA (Vite + Redux Toolkit)
│   ├── src/
│   │   ├── api/                       # HTTP-клиент к API (Axios)
│   │   │   ├── axios.js               # Базовый instance с interceptors
│   │   │   ├── auth.js                # Логин / регистрация
│   │   │   ├── quizzes.js             # CRUD квизов, экспорт/импорт
│   │   │   ├── sessions.js            # Сессии, лидерборд
│   │   │   └── settings.js            # Настройки ведущего
│   │   ├── store/                     # Redux store + slices
│   │   │   ├── index.js               # Конфигурация store
│   │   │   ├── authSlice.js           # Состояние аутентификации
│   │   │   ├── quizSlice.js           # Состояние квизов
│   │   │   └── sessionSlice.js        # Состояние сессий
│   │   ├── pages/                     # Страницы (React Router v6)
│   │   │   ├── LoginPage.jsx          # Страница входа
│   │   │   ├── RegisterPage.jsx       # Страница регистрации
│   │   │   ├── DashboardPage.jsx      # Дашборд — список квизов
│   │   │   ├── QuizEditPage.jsx       # Редактор квиза (категории, вопросы, drag-n-drop)
│   │   │   ├── SessionPage.jsx        # Экран ведущего (лобби → вопросы → лидерборд)
│   │   │   ├── SessionHistoryPage.jsx # История сессий (пагинация, результаты)
│   │   │   └── SettingsPage.jsx       # Настройки (токен бота, ссылка)
│   │   ├── components/                # Переиспользуемые компоненты
│   │   │   ├── Header.jsx             # Шапка с навигацией
│   │   │   ├── ProtectedRoute.jsx     # Защита маршрутов (JWT)
│   │   │   └── QuestionForm.jsx       # Форма вопроса (варианты, цвета, картинки)
│   │   ├── hooks/
│   │   │   └── useWebSocket.js        # Хук для WebSocket-подключения
│   │   ├── App.jsx                    # Роутинг
│   │   ├── App.css                    # Стили (включая mobile responsive)
│   │   ├── index.jsx                  # Точка входа React
│   │   └── index.css                  # Базовые стили
│   ├── index.html
│   ├── nginx.conf                     # Nginx для раздачи SPA + proxy к API/WS/webhook
│   ├── package.json
│   ├── vite.config.js
│   └── Dockerfile
│
├── deploy/                            # Скрипты для деплоя на VPS
│   ├── setup-server.sh                # Первичная настройка сервера (Docker, Nginx, Certbot)
│   └── nginx-site.conf                # Nginx конфигурация для VPS (reverse proxy + SSL)
│
├── docs/                              # Документация проекта
│   ├── PROJECT_OVERVIEW.md            # Обзор, стек, архитектура
│   ├── USER_FLOWS.md                  # Пользовательские сценарии
│   ├── API_DESIGN.md                  # Проектирование API
│   ├── DATABASE_SCHEMA.md             # Схема БД
│   ├── IMPLEMENTATION_PLAN.md         # План реализации
│   └── PROJECT_STRUCTURE.md           # Этот файл
│
├── docker-compose.yml                 # Основной Docker Compose (db, backend, frontend)
├── docker-compose.prod.yml            # Переопределения для production (bind 127.0.0.1)
├── .env                               # Переменные окружения (не в git)
├── .env.example                       # Пример переменных окружения
└── .gitignore
```

---

## Сервисы в Docker Compose

| Сервис     | Порт  | Описание                                 |
|------------|-------|------------------------------------------|
| `db`       | 5432  | PostgreSQL 16                            |
| `backend`  | 8080  | Go API + Telegram Bot Manager            |
| `frontend` | 3000  | React SPA (Nginx в production)           |

## Связи между сервисами

- `backend` → `db` (PostgreSQL через GORM)
- `frontend` → `backend` (HTTP API, WebSocket через Nginx proxy)
- `Telegram` → `backend` (Webhooks через `/webhook/bot/:secret`)

## Переменные окружения (.env)

| Переменная       | Описание                                    |
|------------------|---------------------------------------------|
| `DB_HOST`        | Хост PostgreSQL                             |
| `DB_PORT`        | Порт PostgreSQL                             |
| `DB_USER`        | Пользователь PostgreSQL                     |
| `DB_PASSWORD`    | Пароль PostgreSQL                           |
| `DB_NAME`        | Имя базы данных                             |
| `JWT_SECRET`     | Секрет для JWT-токенов                      |
| `BOT_API_KEY`    | API-ключ для внутренних запросов бота       |
| `WEBHOOK_BASE_URL` | Публичный URL для Telegram webhooks       |
| `POLL_INTERVAL`  | Интервал проверки новых токенов (сек)       |
| `BACKEND_URL`    | URL бэкенда для фронтенда                  |
