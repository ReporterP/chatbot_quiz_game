from aiogram.types import InlineKeyboardMarkup, InlineKeyboardButton, ReplyKeyboardMarkup, KeyboardButton


def main_menu_keyboard() -> ReplyKeyboardMarkup:
    return ReplyKeyboardMarkup(
        keyboard=[
            [KeyboardButton(text="🎮 Войти в квиз")],
            [KeyboardButton(text="👤 Мой профиль"), KeyboardButton(text="📊 История игр")],
        ],
        resize_keyboard=True,
    )


def answer_keyboard(session_id: int, options: list[dict], selected_id: int | None = None) -> InlineKeyboardMarkup:
    buttons = []
    for opt in options:
        text = opt["text"]
        if selected_id and opt["id"] == selected_id:
            text = f"✅ {text}"
        buttons.append([
            InlineKeyboardButton(
                text=text,
                callback_data=f"ans:{session_id}:{opt['id']}",
            )
        ])
    return InlineKeyboardMarkup(inline_keyboard=buttons)
