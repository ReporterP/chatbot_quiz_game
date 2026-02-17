import os
from dotenv import load_dotenv

load_dotenv()

API_BASE_URL = os.getenv("API_BASE_URL", "http://backend:8080")
BOT_API_KEY = os.getenv("BOT_API_KEY", "")
POLL_INTERVAL = float(os.getenv("POLL_INTERVAL", "2"))
TOKEN_REFRESH_INTERVAL = float(os.getenv("TOKEN_REFRESH_INTERVAL", "30"))
