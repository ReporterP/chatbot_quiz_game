import aiohttp
from bot.config import API_BASE_URL, BOT_API_KEY


class ApiClient:
    def __init__(self):
        self._session: aiohttp.ClientSession | None = None

    async def _get_session(self) -> aiohttp.ClientSession:
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession(
                base_url=API_BASE_URL,
                headers={"X-Bot-API-Key": BOT_API_KEY, "Content-Type": "application/json"},
            )
        return self._session

    async def close(self):
        if self._session and not self._session.closed:
            await self._session.close()

    async def get_or_create_user(self, telegram_id: int, nickname: str = "Player") -> dict:
        s = await self._get_session()
        async with s.post("/api/v1/telegram-users", json={
            "telegram_id": telegram_id,
            "nickname": nickname,
        }) as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data

    async def update_nickname(self, telegram_id: int, nickname: str) -> dict:
        s = await self._get_session()
        async with s.put(f"/api/v1/telegram-users/{telegram_id}/nickname", json={
            "nickname": nickname,
        }) as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data

    async def get_history(self, telegram_id: int) -> list:
        s = await self._get_session()
        async with s.get(f"/api/v1/telegram-users/{telegram_id}/history") as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data

    async def join_session(self, code: str, telegram_id: int, nickname: str) -> dict:
        s = await self._get_session()
        async with s.post("/api/v1/sessions/join", json={
            "code": code,
            "telegram_id": telegram_id,
            "nickname": nickname,
        }) as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data

    async def get_session(self, session_id: int) -> dict:
        s = await self._get_session()
        async with s.get(f"/api/v1/sessions/{session_id}") as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data

    async def submit_answer(self, session_id: int, telegram_id: int, option_id: int) -> dict:
        s = await self._get_session()
        async with s.post(f"/api/v1/sessions/{session_id}/answer", json={
            "telegram_id": telegram_id,
            "option_id": option_id,
        }) as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data

    async def get_my_result(self, session_id: int, telegram_id: int) -> dict:
        s = await self._get_session()
        async with s.get(f"/api/v1/sessions/{session_id}/my-result", params={
            "telegram_id": str(telegram_id),
        }) as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data

    async def get_leaderboard(self, session_id: int) -> list:
        s = await self._get_session()
        async with s.get(f"/api/v1/sessions/{session_id}/leaderboard") as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data

    async def get_bot_tokens(self) -> list:
        s = await self._get_session()
        async with s.get("/api/v1/internal/bot-tokens") as r:
            data = await r.json()
            if r.status != 200:
                raise ApiError(data.get("error", "Unknown error"))
            return data


class ApiError(Exception):
    pass
