import logging
import os

from aiogram import Bot, Dispatcher, F, Router
from aiogram.filters import Command, CommandStart
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from aiogram.fsm.storage.memory import MemoryStorage
from aiogram.types import CallbackQuery, Message
from aiogram.utils.keyboard import InlineKeyboardBuilder
from asgiref.sync import sync_to_async

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "quizsite.settings")

import django  # noqa: E402

django.setup()

from quiz.models import AnswerOption, Participant, ParticipantAnswer, QuizSession  # noqa: E402


logging.basicConfig(level=logging.INFO)


class JoinStates(StatesGroup):
    waiting_for_code = State()


router = Router()


@sync_to_async
def _get_session_by_code(code: str):
    return (
        QuizSession.objects.select_related("current_question", "quiz")
        .filter(join_code=code, status=QuizSession.Status.ACTIVE)
        .first()
    )


@sync_to_async
def _get_or_create_participant(session: QuizSession, message: Message):
    participant, _ = Participant.objects.get_or_create(
        session=session,
        telegram_user_id=message.from_user.id,
        defaults={
            "username": message.from_user.username or "",
            "first_name": message.from_user.first_name or "",
            "last_name": message.from_user.last_name or "",
        },
    )
    Participant.objects.filter(pk=participant.id).update(
        username=message.from_user.username or "",
        first_name=message.from_user.first_name or "",
        last_name=message.from_user.last_name or "",
    )
    return participant


@sync_to_async
def _get_current_question(session_id: int):
    session = QuizSession.objects.select_related("current_question").get(pk=session_id)
    return session.current_question


@sync_to_async
def _get_answer_options(question_id: int):
    return list(
        AnswerOption.objects.filter(question_id=question_id).order_by("id").values(
            "id", "text"
        )
    )
@sync_to_async
def _get_active_participant(telegram_user_id: int):
    return (
        Participant.objects.select_related("session")
        .filter(telegram_user_id=telegram_user_id, is_active=True)
        .first()
    )



@sync_to_async
def _answer_exists(participant_id: int, question_id: int):
    return ParticipantAnswer.objects.filter(
        participant_id=participant_id, question_id=question_id
    ).exists()


@sync_to_async
def _create_answer(participant_id: int, question_id: int, option_id: int):
    option = AnswerOption.objects.get(pk=option_id, question_id=question_id)
    answer = ParticipantAnswer(
        participant_id=participant_id,
        question_id=question_id,
        selected_option=option,
    )
    answer.save()
    return option.is_correct




@router.message(CommandStart())
async def start_handler(message: Message, state: FSMContext):
    await message.answer("Привет! Введите код для подключения к квизу.")
    await state.set_state(JoinStates.waiting_for_code)


@router.message(Command("question"))
async def question_handler(message: Message):
    participant = await _get_active_participant(message.from_user.id)
    if not participant:
        await message.answer("Вы еще не подключились к квизу. Используйте /start.")
        return

    question = await _get_current_question(participant.session_id)
    if not question:
        await message.answer("Вопрос еще не задан.")
        return

    await _send_question(message, question.id)


@router.message(JoinStates.waiting_for_code)
async def join_handler(message: Message, state: FSMContext):
    code = (message.text or "").strip()
    session = await _get_session_by_code(code)
    if not session:
        await message.answer("Сессия не найдена или уже завершена. Попробуйте еще раз.")
        return

    participant = await _get_or_create_participant(session, message)
    await state.clear()
    await message.answer(f"Подключение успешно. Ваш код: {session.join_code}")

    await message.answer("Ожидайте появления первого вопроса.")


async def _send_question(message: Message, question_id: int):
    options = await _get_answer_options(question_id)
    if not options:
        await message.answer("Для вопроса пока нет вариантов ответа.")
        return

    builder = InlineKeyboardBuilder()
    for option in options:
        builder.button(
            text=option["text"],
            callback_data=f"answer:{question_id}:{option['id']}",
        )
    builder.adjust(1)
    await message.answer("Выберите вариант ответа:", reply_markup=builder.as_markup())


@router.callback_query(F.data.startswith("answer:"))
async def answer_handler(callback: CallbackQuery):
    parts = callback.data.split(":")
    if len(parts) != 3:
        await callback.answer("Неверные данные.", show_alert=True)
        return
    question_id = int(parts[1])
    option_id = int(parts[2])

    participant = await _get_active_participant(callback.from_user.id)
    if not participant:
        await callback.answer("Сначала подключитесь через /start.", show_alert=True)
        return

    current_question = await _get_current_question(participant.session_id)
    if not current_question or current_question.id != question_id:
        await callback.answer("Этот вопрос уже неактуален.", show_alert=True)
        return

    if await _answer_exists(participant.id, question_id):
        await callback.answer("Ответ уже принят.", show_alert=True)
        return

    is_correct = await _create_answer(participant.id, question_id, option_id)
    if is_correct:
        await callback.answer("Ответ принят! Это верно.", show_alert=False)
    else:
        await callback.answer("Ответ принят!", show_alert=False)


async def main():
    token = os.getenv("BOT_TOKEN")
    if not token:
        raise RuntimeError("BOT_TOKEN is not set")

    bot = Bot(token=token)
    dispatcher = Dispatcher(storage=MemoryStorage())
    dispatcher.include_router(router)

    await dispatcher.start_polling(bot)


if __name__ == "__main__":
    asyncio.run(main())

