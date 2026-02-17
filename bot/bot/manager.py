import asyncio
import logging

from aiogram import Bot, Dispatcher
from aiogram.fsm.storage.memory import MemoryStorage
from aiogram.client.default import DefaultBotProperties

from bot.api_client import ApiClient
from bot.config import TOKEN_REFRESH_INTERVAL
from bot.polling import SessionTracker
from bot.handlers import start, join, game

log = logging.getLogger(__name__)


class BotInstance:
    def __init__(self, token: str, host_id: int, api: ApiClient):
        self.token = token
        self.host_id = host_id
        self.bot = Bot(token=token, default=DefaultBotProperties(parse_mode=None))
        self.storage = MemoryStorage()
        self.dp = Dispatcher(storage=self.storage)
        self.tracker = SessionTracker(api=api, bot=self.bot, storage=self.storage)
        self._task: asyncio.Task | None = None

        self.dp.include_router(start.router)
        self.dp.include_router(join.router)
        self.dp.include_router(game.router)

        @self.dp.update.middleware()
        async def inject_deps(handler, event, data):
            data["api"] = api
            data["tracker"] = self.tracker
            return await handler(event, data)

    async def start(self):
        log.info("Starting bot for host %s", self.host_id)
        self._task = asyncio.create_task(self._run())

    async def _run(self):
        try:
            await self.dp.start_polling(self.bot)
        except asyncio.CancelledError:
            pass
        except Exception as e:
            log.exception("Bot for host %s crashed: %s", self.host_id, e)

    async def stop(self):
        log.info("Stopping bot for host %s", self.host_id)
        if self._task and not self._task.done():
            await self.dp.stop_polling()
            self._task.cancel()
            try:
                await self._task
            except (asyncio.CancelledError, Exception):
                pass
        await self.bot.session.close()


class BotManager:
    def __init__(self, api: ApiClient):
        self.api = api
        self.bots: dict[str, BotInstance] = {}
        self._refresh_task: asyncio.Task | None = None

    async def start(self):
        await self._refresh_tokens()
        self._refresh_task = asyncio.create_task(self._refresh_loop())

    async def stop(self):
        if self._refresh_task and not self._refresh_task.done():
            self._refresh_task.cancel()
            try:
                await self._refresh_task
            except asyncio.CancelledError:
                pass

        for inst in list(self.bots.values()):
            await inst.stop()
        self.bots.clear()

    async def _refresh_loop(self):
        try:
            while True:
                await asyncio.sleep(TOKEN_REFRESH_INTERVAL)
                await self._refresh_tokens()
        except asyncio.CancelledError:
            pass

    async def _refresh_tokens(self):
        try:
            tokens = await self.api.get_bot_tokens()
        except Exception as e:
            log.warning("Failed to fetch bot tokens: %s", e)
            return

        current_tokens = set(self.bots.keys())
        new_tokens = {}
        for entry in tokens:
            token = entry.get("bot_token", "").strip()
            host_id = entry.get("host_id", 0)
            if token:
                new_tokens[token] = host_id

        for token in current_tokens - set(new_tokens.keys()):
            inst = self.bots.pop(token, None)
            if inst:
                await inst.stop()

        for token, host_id in new_tokens.items():
            if token not in self.bots:
                inst = BotInstance(token=token, host_id=host_id, api=self.api)
                self.bots[token] = inst
                await inst.start()

        if new_tokens:
            log.info("Active bots: %d", len(self.bots))
        else:
            log.debug("No bot tokens registered yet")
