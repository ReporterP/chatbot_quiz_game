#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/ReporterP/chatbot_quiz_game.git"
APP_DIR="/opt/chatbot_quiz_game"

echo "=== Quiz Game — Server Setup ==="

# ---------- 1. Docker ----------
if ! command -v docker &>/dev/null; then
  echo ">>> Installing Docker..."
  apt-get update -qq
  apt-get install -y -qq ca-certificates curl gnupg
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin
  systemctl enable --now docker
  echo "Docker installed."
else
  echo "Docker already installed, skipping."
fi

# ---------- 2. Clone repo ----------
if [ ! -d "$APP_DIR" ]; then
  echo ">>> Cloning repository..."
  git clone "$REPO_URL" "$APP_DIR"
else
  echo "Repository already cloned at $APP_DIR, pulling latest..."
  cd "$APP_DIR" && git pull origin main
fi

# ---------- 3. .env ----------
if [ ! -f "$APP_DIR/.env" ]; then
  cp "$APP_DIR/.env.example" "$APP_DIR/.env"
  echo ">>> Created .env from template. EDIT IT before starting:"
  echo "    nano $APP_DIR/.env"
  echo ""
  echo "    Required values to set:"
  echo "    - JWT_SECRET (random string)"
  echo "    - BOT_API_KEY (random string)"
  echo "    - DB_PASSWORD (secure password)"
  echo "    - WEBHOOK_BASE_URL (https://yourdomain.com)"
else
  echo ".env already exists, skipping."
fi

# ---------- 4. Nginx + SSL ----------
if ! command -v nginx &>/dev/null; then
  echo ">>> Installing Nginx..."
  apt-get install -y -qq nginx
  systemctl enable --now nginx
fi

if ! command -v certbot &>/dev/null; then
  echo ">>> Installing Certbot..."
  apt-get install -y -qq certbot python3-certbot-nginx
fi

echo ""
echo "=== Setup complete ==="
echo ""
echo "Next steps:"
echo "  1. Edit .env:  nano $APP_DIR/.env"
echo "  2. Create Nginx site config:  nano /etc/nginx/sites-available/quizgame"
echo "     (see deploy/nginx-site.conf in the repo for a template)"
echo "  3. Enable site:  ln -sf /etc/nginx/sites-available/quizgame /etc/nginx/sites-enabled/"
echo "  4. Remove default:  rm -f /etc/nginx/sites-enabled/default"
echo "  5. Get SSL:  certbot --nginx -d yourdomain.com"
echo "  6. Start app:  cd $APP_DIR && docker compose up --build -d"
echo ""
