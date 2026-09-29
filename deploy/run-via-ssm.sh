#!/bin/bash
# Sends a local shell script to run on the production instance via SSM
# RunCommand, streams its status, and exits non-zero (failing the GitHub
# Actions job) if the remote script fails or the command itself errors.
#
# Usage: run-via-ssm.sh <script-file> <instance-id> <comment>
set -euo pipefail

SCRIPT_FILE="$1"
INSTANCE_ID="$2"
COMMENT="$3"

PARAMS_FILE=$(mktemp)
trap 'rm -f "$PARAMS_FILE"' EXIT
jq -Rn --rawfile script "$SCRIPT_FILE" '{commands: ($script | rtrimstr("\n") | split("\n"))}' > "$PARAMS_FILE"

CID=$(aws ssm send-command \
  --instance-ids "$INSTANCE_ID" \
  --document-name AWS-RunShellScript \
  --comment "$COMMENT" \
  --timeout-seconds 600 \
  --parameters "file://$PARAMS_FILE" \
  --query 'Command.CommandId' --output text)
echo "SSM CommandId: $CID"

# Give SSM a moment to register the command before the first poll.
sleep 2

for _ in $(seq 1 90); do
  STATUS=$(aws ssm get-command-invocation --command-id "$CID" --instance-id "$INSTANCE_ID" --query 'Status' --output text 2>/dev/null || echo "Pending")
  echo "status=$STATUS"
  case "$STATUS" in
    Success)
      aws ssm get-command-invocation --command-id "$CID" --instance-id "$INSTANCE_ID" --query 'StandardOutputContent' --output text
      exit 0
      ;;
    Failed | Cancelled | TimedOut)
      echo "--- stdout ---"
      aws ssm get-command-invocation --command-id "$CID" --instance-id "$INSTANCE_ID" --query 'StandardOutputContent' --output text
      echo "--- stderr ---"
      aws ssm get-command-invocation --command-id "$CID" --instance-id "$INSTANCE_ID" --query 'StandardErrorContent' --output text
      exit 1
      ;;
  esac
  sleep 5
done

echo "Timed out waiting for SSM command $CID to finish"
exit 1
