import asyncio
import logging
from dataclasses import dataclass, field

from aiogram import Bot
from aiogram.fsm.storage.base import StorageKey
from aiogram.fsm.storage.memory import MemoryStorage

from bot.api_client import ApiClient, ApiError
from bot.keyboards import answer_keyboard
from bot.config import POLL_INTERVAL

log = logging.getLogger(__name__)


@dataclass
class ParticipantInfo:
    chat_id: int
    telegram_id: int
    message_id: int | None = None
    selected_option_id: int | None = None


@dataclass
class SessionInfo:
    session_id: int
    last_status: str = ""
    last_question: int = 0
    participants: dict[int, ParticipantInfo] = field(default_factory=dict)


class SessionTracker:
    def __init__(self, api: ApiClient, bot: Bot, storage: MemoryStorage):
        self.api = api
        self.bot = bot
        self.storage = storage
        self.sessions: dict[int, SessionInfo] = {}
        self._tasks: dict[int, asyncio.Task] = {}

    async def add_participant(self, session_id: int, telegram_id: int, chat_id: int):
        if session_id not in self.sessions:
            self.sessions[session_id] = SessionInfo(session_id=session_id)
            task = asyncio.create_task(self._poll_loop(session_id))
            self._tasks[session_id] = task

        self.sessions[session_id].participants[telegram_id] = ParticipantInfo(
            chat_id=chat_id,
            telegram_id=telegram_id,
        )

    def _remove_session(self, session_id: int):
        self.sessions.pop(session_id, None)
        task = self._tasks.pop(session_id, None)
        if task and not task.done():
            task.cancel()

    async def _poll_loop(self, session_id: int):
        try:
            while session_id in self.sessions:
                await self._check_session(session_id)
                await asyncio.sleep(POLL_INTERVAL)
        except asyncio.CancelledError:
            pass
        except Exception as e:
            log.exception("Poll loop error for session %s: %s", session_id, e)

    async def _check_session(self, session_id: int):
        info = self.sessions.get(session_id)
        if not info:
            return

        try:
            state = await self.api.get_session(session_id)
        except ApiError:
            return

        status = state.get("status", "")
        current_q = state.get("current_question", 0)
        question_data = state.get("current_question_data")

        if info.last_status == status and info.last_question == current_q:
            return

        prev_status = info.last_status
        prev_q = info.last_question
        info.last_status = status
        info.last_question = current_q

        if status == "question" and question_data and current_q != prev_q:
            await self._send_question(info, state, question_data)

        elif status == "revealed" and prev_status == "question":
            await self._send_results(info, state)

        elif status == "finished" and prev_status != "finished":
            await self._send_leaderboard(info)
            self._remove_session(session_id)

    async def _send_question(self, info: SessionInfo, state: dict, question_data: dict):
        current = state.get("current_question", 0)
        total = state.get("total_questions", 0)
        text = (
            f"❓ <b>Вопрос {current} из {total}</b>\n\n"
            f"{question_data['text']}"
        )
        kb = answer_keyboard(info.session_id, question_data.get("options", []))

        for tg_id, pinfo in info.participants.items():
            pinfo.selected_option_id = None
            try:
                if pinfo.message_id:
                    try:
                        await self.bot.edit_message_text(
                            text, chat_id=pinfo.chat_id, message_id=pinfo.message_id,
                            reply_markup=kb, parse_mode="HTML",
                        )
                        await self._update_fsm(pinfo, info.session_id, question_data, current, total)
                        continue
                    except Exception:
                        pass

                msg = await self.bot.send_message(pinfo.chat_id, text, reply_markup=kb, parse_mode="HTML")
                pinfo.message_id = msg.message_id
                await self._update_fsm(pinfo, info.session_id, question_data, current, total)
            except Exception as e:
                log.warning("Failed to send question to %s: %s", pinfo.chat_id, e)

    async def _update_fsm(self, pinfo: ParticipantInfo, session_id: int, question_data: dict, current: int, total: int):
        key = StorageKey(bot_id=self.bot.id, chat_id=pinfo.chat_id, user_id=pinfo.telegram_id)
        await self.storage.update_data(key=key, data={
            "current_question_data": question_data,
            "current_q_num": current,
            "total_questions": total,
            "selected_option_id": None,
        })

    async def _send_results(self, info: SessionInfo, state: dict):
        question_data = state.get("current_question_data")
        current = state.get("current_question", 0)
        total = state.get("total_questions", 0)

        for tg_id, pinfo in info.participants.items():
            try:
                result = await self.api.get_my_result(info.session_id, tg_id)
            except ApiError:
                continue

            if not result.get("answered"):
                result_line = "⏰ Вы не успели ответить"
                score_line = ""
            elif result.get("is_correct"):
                score = result.get("score", 0)
                total_score = result.get("total_score", 0)
                result_line = "✅ <b>Правильно!</b>"
                score_line = f"\nОчки за вопрос: <b>+{score}</b> | Всего: <b>{total_score}</b>"
            else:
                total_score = result.get("total_score", 0)
                result_line = "❌ <b>Неправильно</b>"
                score_line = f"\nВсего очков: <b>{total_score}</b>"

            correct_text = ""
            if question_data:
                for opt in question_data.get("options", []):
                    if opt.get("is_correct"):
                        correct_text = f"\n\nПравильный ответ: <b>{opt['text']}</b>"
                        break

            text = (
                f"❓ <b>Вопрос {current} из {total}</b>\n\n"
                f"{question_data['text'] if question_data else ''}\n\n"
                f"{result_line}{score_line}{correct_text}\n\n"
                f"⏳ Ожидайте следующий вопрос..."
            )

            try:
                if pinfo.message_id:
                    await self.bot.edit_message_text(
                        text, chat_id=pinfo.chat_id, message_id=pinfo.message_id,
                        parse_mode="HTML",
                    )
                else:
                    msg = await self.bot.send_message(pinfo.chat_id, text, parse_mode="HTML")
                    pinfo.message_id = msg.message_id
            except Exception as e:
                log.warning("Failed to send result to %s: %s", pinfo.chat_id, e)

    async def _send_leaderboard(self, info: SessionInfo):
        try:
            entries = await self.api.get_leaderboard(info.session_id)
        except ApiError:
            return

        lines = ["🏆 <b>Квиз завершён! Итоги:</b>\n"]
        medals = {1: "🥇", 2: "🥈", 3: "🥉"}
        for e in entries:
            pos = e["position"]
            medal = medals.get(pos, f"{pos}.")
            lines.append(f"{medal} <b>{e['nickname']}</b> — {e['total_score']} очков")

        base_text = "\n".join(lines)

        for tg_id, pinfo in info.participants.items():
            my_pos = None
            for e in entries:
                if e.get("telegram_id") == tg_id:
                    my_pos = e["position"]
                    break

            personal = base_text
            if my_pos is not None:
                personal += f"\n\n📍 Ваше место: <b>{my_pos}</b>"
            personal += "\n\nДля новой игры нажмите /start"

            try:
                if pinfo.message_id:
                    await self.bot.edit_message_text(
                        personal, chat_id=pinfo.chat_id, message_id=pinfo.message_id,
                        parse_mode="HTML",
                    )
                else:
                    await self.bot.send_message(pinfo.chat_id, personal, parse_mode="HTML")
            except Exception as e:
                log.warning("Failed to send leaderboard to %s: %s", pinfo.chat_id, e)

            key = StorageKey(bot_id=self.bot.id, chat_id=pinfo.chat_id, user_id=tg_id)
            await self.storage.update_data(key=key, data={})
            await self.storage.set_state(key=key, state=None)
