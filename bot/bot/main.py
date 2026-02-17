import asyncio
import logging

from bot.api_client import ApiClient
from bot.manager import BotManager

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
log = logging.getLogger(__name__)


async def main():
    api = ApiClient()
    manager = BotManager(api=api)

    log.info("Bot manager starting...")
    try:
        await manager.start()
        while True:
            await asyncio.sleep(1)
    except (KeyboardInterrupt, asyncio.CancelledError):
        pass
    finally:
        await manager.stop()
        await api.close()
        log.info("Bot manager stopped.")


if __name__ == "__main__":
    asyncio.run(main())
