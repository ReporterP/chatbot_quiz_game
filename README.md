# chatbot_quiz_game

Минимальный проект Django + aiogram для квиз-игры.

## Запуск
1. Установить зависимости: `pip install -r requirements.txt`
2. Настроить переменные окружения:
   - `DJANGO_SECRET_KEY`
   - `DJANGO_DEBUG` (1/0)
   - `DJANGO_ALLOWED_HOSTS` (через запятую)
   - `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`
   - `BOT_TOKEN`
3. Миграции и админ:
   - `python manage.py migrate`
   - `python manage.py createsuperuser`
4. Запуск web: `python manage.py runserver`
5. Запуск бота: `python bot/run_bot.py`

## Поток работы
- В админке создайте квиз, вопросы и варианты ответа.
- В web панели создайте сессию — появится код подключения.
- Участники вводят код в боте и отвечают на вопросы.
- На экране `/screen/<код>` показываются вопрос, ответ и таблица лидеров.