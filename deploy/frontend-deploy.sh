#!/bin/bash
# Runs ON the production EC2 instance via SSM RunCommand (never locally).
# Builds the frontend from the latest main and swaps it into nginx, with an
# automatic rollback if the site doesn't come back healthy.
set -e

echo "--- updating build checkout to latest main ---"
cd /opt/campusdesk-build
git fetch origin
git reset --hard origin/main
git log --oneline -3

echo "--- building frontend ---"
cd /opt/campusdesk-build/frontend
npm ci --no-audit --no-fund --silent
npm run build

echo "--- swap: frontend ---"
TS=$(date +%Y%m%d%H%M%S)
mkdir -p /opt/nginx-html-backups
cp -a /usr/share/nginx/html /opt/nginx-html-backups/html.bak-$TS
rsync -a --delete /opt/campusdesk-build/frontend/dist/ /usr/share/nginx/html/
nginx -t
systemctl reload nginx

FCODE=$(curl -sk -o /dev/null -w "%{http_code}" --max-time 5 -H "Host: 13-202-93-187.sslip.io" https://127.0.0.1/)
echo "frontend https status: $FCODE"

if [ "$FCODE" != "200" ]; then
  echo DEPLOY_FAILED_FRONTEND_HEALTH_CHECK
  rm -rf /usr/share/nginx/html
  cp -a /opt/nginx-html-backups/html.bak-$TS /usr/share/nginx/html
  systemctl reload nginx
  echo FRONTEND_ROLLED_BACK
  exit 1
fi

echo FRONTEND_DEPLOY_COMPLETE ts=$TS
