#!/bin/bash
# One-time setup: lets GitHub Actions deploy this repo without any long-lived
# AWS keys. It creates (if not already present) the GitHub OIDC identity
# provider, then a narrowly-scoped IAM role that only GitHub Actions runs of
# ajaypatel01/CampusDesk can assume, and that can only call
# ssm:SendCommand against the one production instance plus
# ssm:GetCommandInvocation (which AWS doesn't let you scope to a resource).
#
# Run this with credentials that have IAM admin rights -- the credentials
# this session has been using for SSM/RDS work do NOT have IAM permissions,
# so this has to be run by you or whoever manages this AWS account.
#
# Safe to re-run: every step checks for the existing resource first.
set -euo pipefail

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ROLE_NAME="campusdesk-github-deploy"
POLICY_NAME="campusdesk-github-deploy-policy"
REPO="ajaypatel01/CampusDesk"
REGION="ap-south-1"
INSTANCE_ID="i-058bf81226520c730"
OIDC_URL="token.actions.githubusercontent.com"
OIDC_ARN="arn:aws:iam::${ACCOUNT_ID}:oidc-provider/${OIDC_URL}"

echo "Account: $ACCOUNT_ID"

echo "--- OIDC provider ---"
if aws iam get-open-id-connect-provider --open-id-connect-provider-arn "$OIDC_ARN" >/dev/null 2>&1; then
  echo "already exists: $OIDC_ARN"
else
  aws iam create-open-id-connect-provider \
    --url "https://${OIDC_URL}" \
    --client-id-list "sts.amazonaws.com" \
    --thumbprint-list "6938fd4d98bab03faadb97b34396831e3780aea1"
  echo "created: $OIDC_ARN"
fi

echo "--- trust policy ---"
TRUST_POLICY_FILE=$(mktemp)
trap 'rm -f "$TRUST_POLICY_FILE"' EXIT
cat > "$TRUST_POLICY_FILE" <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Federated": "${OIDC_ARN}" },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": { "${OIDC_URL}:aud": "sts.amazonaws.com" },
        "StringLike": { "${OIDC_URL}:sub": "repo:${REPO}:*" }
      }
    }
  ]
}
EOF

echo "--- permissions policy (least privilege: one instance, SSM only) ---"
PERMS_POLICY_FILE=$(mktemp)
trap 'rm -f "$PERMS_POLICY_FILE"' EXIT
cat > "$PERMS_POLICY_FILE" <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "ssm:SendCommand",
      "Resource": [
        "arn:aws:ec2:${REGION}:${ACCOUNT_ID}:instance/${INSTANCE_ID}",
        "arn:aws:ssm:${REGION}::document/AWS-RunShellScript"
      ]
    },
    {
      "Effect": "Allow",
      "Action": "ssm:GetCommandInvocation",
      "Resource": "*"
    }
  ]
}
EOF

echo "--- role ---"
if aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
  echo "role exists, updating trust policy"
  aws iam update-assume-role-policy --role-name "$ROLE_NAME" --policy-document "file://$TRUST_POLICY_FILE"
else
  aws iam create-role \
    --role-name "$ROLE_NAME" \
    --assume-role-policy-document "file://$TRUST_POLICY_FILE" \
    --description "GitHub Actions deploy role for ${REPO} -- SSM-only, one instance"
fi

aws iam put-role-policy \
  --role-name "$ROLE_NAME" \
  --policy-name "$POLICY_NAME" \
  --policy-document "file://$PERMS_POLICY_FILE"

ROLE_ARN=$(aws iam get-role --role-name "$ROLE_NAME" --query 'Role.Arn' --output text)
echo ""
echo "=================================================================="
echo "Done. Add this as a GitHub Actions repo secret named AWS_DEPLOY_ROLE_ARN:"
echo ""
echo "  $ROLE_ARN"
echo ""
echo "Via the CLI (needs gh auth login first):"
echo "  gh secret set AWS_DEPLOY_ROLE_ARN --repo ${REPO} --body \"$ROLE_ARN\""
echo ""
echo "Or via the web UI:"
echo "  https://github.com/${REPO}/settings/secrets/actions/new"
echo "=================================================================="
