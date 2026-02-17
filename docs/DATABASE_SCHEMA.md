# Quiz Game — Схема базы данных

## ER-диаграмма (текстовая)

```
hosts ──< quizzes ──< categories ──< questions ──< options
                                        │
                                   question_images
                        │
                    sessions ──< participants
                        │              │
                        └──< answers ──┘

telegram_users (привязка ника к Telegram ID)
```

---

## Таблицы

### hosts (ведущие)
| Поле          | Тип          | Описание                     |
|---------------|--------------|------------------------------|
| id            | BIGSERIAL PK | ID ведущего                  |
| username      | VARCHAR(100) | Логин (уникальный)           |
| password_hash | VARCHAR(255) | Хеш пароля (bcrypt)          |
| bot_token     | VARCHAR(255) | Токен Telegram-бота ведущего |
| bot_link      | VARCHAR(255) | Ссылка на Telegram-бота      |
| created_at    | TIMESTAMP    | Дата регистрации             |

---

### telegram_users (пользователи Telegram)
| Поле         | Тип          | Описание                     |
|--------------|--------------|------------------------------|
| id           | BIGSERIAL PK | ID                           |
| telegram_id  | BIGINT       | Telegram ID (уникальный)     |
| nickname     | VARCHAR(100) | Сохранённый ник              |
| created_at   | TIMESTAMP    | Дата создания                |
| updated_at   | TIMESTAMP    | Дата обновления              |

---

### quizzes (квизы)
| Поле       | Тип          | Описание                     |
|------------|--------------|------------------------------|
| id         | BIGSERIAL PK | ID квиза                     |
| host_id    | BIGINT FK    | Ведущий-автор                |
| title      | VARCHAR(255) | Название квиза               |
| created_at | TIMESTAMP    | Дата создания                |
| updated_at | TIMESTAMP    | Дата обновления              |

---

### categories (категории вопросов)
| Поле       | Тип          | Описание                     |
|------------|--------------|------------------------------|
| id         | BIGSERIAL PK | ID категории                 |
| quiz_id    | BIGINT FK    | Квиз                         |
| title      | VARCHAR(255) | Название категории           |
| order_num  | INT          | Порядковый номер             |

---

### questions (вопросы)
| Поле        | Тип          | Описание                     |
|-------------|--------------|------------------------------|
| id          | BIGSERIAL PK | ID вопроса                   |
| quiz_id     | BIGINT FK    | Квиз                         |
| category_id | BIGINT FK    | Категория (nullable)         |
| text        | TEXT         | Текст вопроса                |
| order_num   | INT          | Порядковый номер в категории |

---

### question_images (картинки вопросов)
| Поле        | Тип          | Описание                     |
|-------------|--------------|------------------------------|
| id          | BIGSERIAL PK | ID                           |
| question_id | BIGINT FK    | Вопрос                       |
| url         | VARCHAR(500) | Путь к файлу                 |
| order_num   | INT          | Порядок отображения          |

---

### options (варианты ответа)
| Поле        | Тип          | Описание                     |
|-------------|--------------|------------------------------|
| id          | BIGSERIAL PK | ID варианта                  |
| question_id | BIGINT FK    | Вопрос                       |
| text        | VARCHAR(500) | Текст варианта               |
| is_correct  | BOOLEAN      | Правильный ли вариант        |
| color       | VARCHAR(7)   | HEX-цвет варианта            |

---

### sessions (сессии квиза)
| Поле              | Тип           | Описание                                    |
|-------------------|---------------|---------------------------------------------|
| id                | BIGSERIAL PK  | ID сессии                                   |
| quiz_id           | BIGINT FK     | Квиз                                        |
| host_id           | BIGINT FK     | Ведущий                                     |
| code              | VARCHAR(6)    | Код для подключения                         |
| status            | VARCHAR(20)   | waiting / question / revealed / finished    |
| current_question  | INT           | Индекс текущего вопроса                     |
| created_at        | TIMESTAMP     | Дата создания                               |

---

### participants (участники)
| Поле         | Тип          | Описание                        |
|--------------|--------------|---------------------------------|
| id           | BIGSERIAL PK | ID участника                    |
| session_id   | BIGINT FK    | Сессия                          |
| telegram_id  | BIGINT       | Telegram ID                     |
| nickname     | VARCHAR(100) | Ник                             |
| total_score  | INT          | Суммарные очки                  |
| joined_at    | TIMESTAMP    | Время присоединения             |

---

### answers (ответы)
| Поле           | Тип          | Описание                             |
|----------------|--------------|--------------------------------------|
| id             | BIGSERIAL PK | ID ответа                            |
| session_id     | BIGINT FK    | Сессия                               |
| participant_id | BIGINT FK    | Участник                             |
| question_id    | BIGINT FK    | Вопрос                               |
| option_id      | BIGINT FK    | Выбранный вариант                    |
| is_correct     | BOOLEAN      | Правильный ли ответ                  |
| score          | INT          | Начисленные очки                     |
| answered_at    | TIMESTAMP    | Время ответа                         |

**Изменение ответа**: участник может менять ответ пока статус сессии = `question`.
