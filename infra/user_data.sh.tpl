#!/bin/bash
# Bootstraps Docker, the app directory + .env, and nightly Postgres backups.
set -euxo pipefail

dnf install -y docker cronie
systemctl enable --now docker crond
usermod -aG docker ec2-user

# docker compose v2 plugin
mkdir -p /usr/local/lib/docker/cli-plugins
curl -fsSL "https://github.com/docker/compose/releases/latest/download/docker-compose-linux-$(uname -m)" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

APP_DIR=/home/ec2-user/booker
mkdir -p "$APP_DIR"

umask 077
cat > "$APP_DIR/.env" <<'ENVEOF'
DB_USER=${db_user}
DB_PASSWORD=${db_password}
DB_NAME=${db_name}
DISCORD_TOKEN=${discord_token}
DISCORD_GUILD_ID=${discord_guild_id}
ENVEOF
chown -R ec2-user:ec2-user "$APP_DIR"
chmod 600 "$APP_DIR/.env"

# Nightly backup at 03:00 UTC (no-op until the stack has been deployed once)
cat > /etc/cron.d/booker-backup <<'CRONEOF'
0 3 * * * ec2-user cd /home/ec2-user/booker && [ -f compose.prod.yaml ] && docker compose -f compose.prod.yaml exec -T db pg_dump -U ${db_user} ${db_name} | gzip | aws s3 cp - s3://${backup_bucket}/booker-$(date +\%F).sql.gz
CRONEOF
chmod 644 /etc/cron.d/booker-backup
