from aiogram import Router, types
from aiogram.fsm.context import FSMContext

from bot.states import QuizStates
from bot.api_client import ApiClient, ApiError
from bot.keyboards import main_menu_keyboard

router = Router()


@router.message(QuizStates.enter_code)
async def on_code(message: types.Message, state: FSMContext, api: ApiClient, tracker):
    code = message.text.strip()

    if not code.isdigit() or len(code) != 6:
        await message.answer("❌ Код должен состоять из 6 цифр. Попробуйте ещё раз:")
        return

    try:
        result = await api.get_or_create_user(message.from_user.id, message.from_user.first_name or "Player")
        user = result["user"]
        nickname = user["nickname"]
        created = result.get("created", False)
    except ApiError:
        nickname = None
        created = True

    if nickname and not created:
        await state.update_data(code=code, nickname=nickname)
        await _join_session(message, state, api, code, nickname, tracker)
    else:
        await state.update_data(code=code)
        await state.set_state(QuizStates.enter_nickname)
        await message.answer(
            f"✅ Код принят: <b>{code}</b>\n\nВведите ваш никнейм:",
            parse_mode="HTML",
        )


@router.message(QuizStates.enter_nickname)
async def on_nickname(message: types.Message, state: FSMContext, api: ApiClient, tracker):
    nickname = message.text.strip()

    if len(nickname) < 1 or len(nickname) > 100:
        await message.answer("❌ Никнейм должен быть от 1 до 100 символов. Попробуйте ещё раз:")
        return

    try:
        await api.update_nickname(message.from_user.id, nickname)
    except ApiError:
        try:
            await api.get_or_create_user(message.from_user.id, nickname)
        except ApiError:
            pass

    data = await state.get_data()
    code = data.get("code", "")

    if not code:
        await state.clear()
        await message.answer(
            f"✅ Никнейм установлен: <b>{nickname}</b>\n\nВыберите действие:",
            parse_mode="HTML",
            reply_markup=main_menu_keyboard(),
        )
        return

    await _join_session(message, state, api, code, nickname, tracker)


async def _join_session(message, state, api, code, nickname, tracker=None):
    try:
        result = await api.join_session(code, message.from_user.id, nickname)
    except ApiError as e:
        await message.answer(
            f"❌ Ошибка: {e}\n\nПопробуйте /start заново.",
            reply_markup=main_menu_keyboard(),
        )
        await state.clear()
        return

    session_id = result["session_id"]
    await state.update_data(session_id=session_id, nickname=nickname)
    await state.set_state(QuizStates.in_session)

    msg = await message.answer(
        f"🎮 Вы подключились к квизу!\n\n"
        f"Никнейм: <b>{nickname}</b>\n"
        f"Ожидайте начала игры...",
        parse_mode="HTML",
    )

    if tracker:
        await tracker.add_participant(session_id, message.from_user.id, message.chat.id)
        info = tracker.sessions.get(session_id)
        if info and message.from_user.id in info.participants:
            info.participants[message.from_user.id].message_id = msg.message_id
