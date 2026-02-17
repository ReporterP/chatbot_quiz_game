from aiogram import Router, types, F
from aiogram.filters import CommandStart, CommandObject
from aiogram.fsm.context import FSMContext

from bot.states import QuizStates
from bot.api_client import ApiClient, ApiError
from bot.keyboards import main_menu_keyboard

router = Router()


@router.message(CommandStart())
async def cmd_start(message: types.Message, state: FSMContext, api: ApiClient, command: CommandObject, tracker):
    await state.clear()

    try:
        result = await api.get_or_create_user(message.from_user.id, message.from_user.first_name or "Player")
        user = result["user"]
        nickname = user["nickname"]
        created = result.get("created", False)
    except ApiError:
        nickname = None
        created = True

    if command.args:
        code = command.args.strip()
        if nickname and not created:
            await state.update_data(code=code, nickname=nickname)
            await _do_join(message, state, api, code, nickname, tracker)
        else:
            await state.update_data(code=code)
            await state.set_state(QuizStates.enter_nickname)
            await message.answer(
                f"👋 Добро пожаловать в Quiz Game!\n\n"
                f"Код сессии: <b>{code}</b>\n"
                f"Введите ваш никнейм:",
                parse_mode="HTML",
            )
        return

    if nickname and not created:
        await message.answer(
            f"👋 Привет, <b>{nickname}</b>!\n\n"
            f"Выберите действие:",
            parse_mode="HTML",
            reply_markup=main_menu_keyboard(),
        )
    else:
        await state.set_state(QuizStates.enter_nickname)
        await message.answer(
            "👋 Добро пожаловать в Quiz Game!\n\n"
            "Введите ваш никнейм:",
        )


@router.message(F.text == "🎮 Войти в квиз")
async def on_join_button(message: types.Message, state: FSMContext):
    await state.set_state(QuizStates.enter_code)
    await message.answer("Введите 6-значный код сессии:")


@router.message(F.text == "👤 Мой профиль")
async def on_profile(message: types.Message, api: ApiClient):
    try:
        result = await api.get_or_create_user(message.from_user.id)
        user = result["user"]
        await message.answer(
            f"👤 <b>Ваш профиль</b>\n\n"
            f"Никнейм: <b>{user['nickname']}</b>\n\n"
            f"Чтобы изменить ник, отправьте:\n/nickname Новый_ник",
            parse_mode="HTML",
        )
    except ApiError as e:
        await message.answer(f"Ошибка: {e}")


@router.message(F.text == "📊 История игр")
async def on_history(message: types.Message, api: ApiClient):
    try:
        entries = await api.get_history(message.from_user.id)
    except ApiError:
        entries = []

    if not entries:
        await message.answer("📊 У вас пока нет завершённых игр.")
        return

    lines = ["📊 <b>Ваша история игр:</b>\n"]
    medals = {1: "🥇", 2: "🥈", 3: "🥉"}
    for e in entries[:20]:
        pos = e.get("position", 0)
        medal = medals.get(pos, f"{pos}.")
        lines.append(
            f"{medal} <b>{e['quiz_title']}</b>\n"
            f"   Очки: {e['total_score']} | Место: {pos}/{e['total_players']}"
        )

    await message.answer("\n".join(lines), parse_mode="HTML")


@router.message(F.text.startswith("/nickname"))
async def on_change_nickname(message: types.Message, api: ApiClient):
    parts = message.text.split(maxsplit=1)
    if len(parts) < 2 or not parts[1].strip():
        await message.answer("Использование: /nickname Ваш_новый_ник")
        return

    new_nick = parts[1].strip()
    if len(new_nick) > 100:
        await message.answer("Никнейм слишком длинный (макс 100 символов)")
        return

    try:
        user = await api.update_nickname(message.from_user.id, new_nick)
        await message.answer(
            f"✅ Никнейм изменён на: <b>{user['nickname']}</b>",
            parse_mode="HTML",
            reply_markup=main_menu_keyboard(),
        )
    except ApiError as e:
        await message.answer(f"Ошибка: {e}")


async def _do_join(message, state, api, code, nickname, tracker=None):
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
