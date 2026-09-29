#!/bin/bash
# Runs ON the production EC2 instance via SSM RunCommand (never locally).
# Builds the Go backend from the latest main and hot-swaps the running
# service, with an automatic rollback if it doesn't come back healthy.
#
# Deliberately does NOT run database migrations -- those stay a separate,
# explicitly-requested manual step (see deploy/README.md). Auto-applying a
# schema change on every merge is a different, much riskier kind of pipeline
# than "redeploy the code that's already been tested against the DB."
set -e

export HOME=/root
export GOPATH=/root/go
export GOCACHE=/root/.cache/go-build
export PATH=$PATH:/usr/local/go/bin
mkdir -p "$GOPATH" "$GOCACHE"

echo "--- updating build checkout to latest main ---"
cd /opt/campusdesk-build
git fetch origin
git reset --hard origin/main
git log --oneline -3

echo "--- building backend ---"
go build -ldflags="-s -w" -o /opt/campusdesk-build/campusdesk-new ./cmd/server
file /opt/campusdesk-build/campusdesk-new

echo "--- swap: backend ---"
TS=$(date +%Y%m%d%H%M%S)
cp -a /opt/campusdesk/campusdesk /opt/campusdesk/campusdesk.bak-$TS
systemctl stop campusdesk
install -o ec2-user -g ec2-user -m 0755 /opt/campusdesk-build/campusdesk-new /opt/campusdesk/campusdesk
systemctl start campusdesk
sleep 3

if ! systemctl is-active --quiet campusdesk; then
  echo DEPLOY_FAILED_SERVICE_INACTIVE
  journalctl -u campusdesk --no-pager -n 40
  systemctl stop campusdesk
  cp -a /opt/campusdesk/campusdesk.bak-$TS /opt/campusdesk/campusdesk
  systemctl start campusdesk
  echo BACKEND_ROLLED_BACK
  exit 1
fi

CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 http://127.0.0.1:8080/api/v1/ready)
echo "backend /ready http status: $CODE"
DBERR=$(journalctl -u campusdesk --no-pager -n 30 | grep -ciE "failed to connect|connection refused|no such host|dial tcp.*timeout" || true)
echo "db_error_lines_in_recent_log: $DBERR"

if [ "$CODE" != "200" ] || [ "$DBERR" -gt 0 ]; then
  echo DEPLOY_FAILED_HEALTH_CHECK
  journalctl -u campusdesk --no-pager -n 40
  systemctl stop campusdesk
  cp -a /opt/campusdesk/campusdesk.bak-$TS /opt/campusdesk/campusdesk
  systemctl start campusdesk
  echo BACKEND_ROLLED_BACK
  exit 1
fi

echo BACKEND_DEPLOY_COMPLETE ts=$TS
