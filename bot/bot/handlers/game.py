from aiogram import Router, types, F
from aiogram.fsm.context import FSMContext

from bot.states import QuizStates
from bot.api_client import ApiClient, ApiError
from bot.keyboards import answer_keyboard

router = Router()


@router.callback_query(F.data.startswith("ans:"))
async def on_answer(callback: types.CallbackQuery, state: FSMContext, api: ApiClient):
    current = await state.get_state()
    if current != QuizStates.in_session.state:
        await callback.answer("Вы не в активной сессии", show_alert=True)
        return

    parts = callback.data.split(":")
    if len(parts) != 3:
        await callback.answer("Неверные данные", show_alert=True)
        return

    session_id = int(parts[1])
    option_id = int(parts[2])

    try:
        await api.submit_answer(session_id, callback.from_user.id, option_id)
    except ApiError as e:
        error_text = str(e)
        if "not accepting" in error_text:
            await callback.answer("Время для ответа вышло", show_alert=True)
        else:
            await callback.answer(f"Ошибка: {error_text}", show_alert=True)
        return

    data = await state.get_data()
    question_data = data.get("current_question_data")
    if not question_data:
        await callback.answer("✅ Ответ принят!")
        return

    kb = answer_keyboard(session_id, question_data.get("options", []), selected_id=option_id)

    current_q = data.get("current_q_num", 0)
    total = data.get("total_questions", 0)
    text = (
        f"❓ <b>Вопрос {current_q} из {total}</b>\n\n"
        f"{question_data['text']}\n\n"
        f"✅ <b>Ваш ответ принят</b>"
    )

    try:
        await callback.message.edit_text(text, reply_markup=kb, parse_mode="HTML")
    except Exception:
        pass

    await state.update_data(selected_option_id=option_id)
    await callback.answer("✅ Ответ принят!")
