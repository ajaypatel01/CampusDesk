# Deployment pipeline

GitHub Actions workflow at `.github/workflows/deploy.yml`. On every push to
`main`, it deploys whichever of the backend / frontend actually changed
(path-filtered), or you can trigger either (or both) manually from the
Actions tab with the "Run workflow" button.

Both jobs do exactly what the manual SSM deploy scripts used earlier in this
project's history did by hand: pull `origin/main` on the instance's
`/opt/campusdesk-build` checkout, build in place, hot-swap the running
service/nginx files, health-check, and **automatically roll back** to the
previous binary/`dist/` if the health check fails.

## What this pipeline deliberately does NOT do

**It never runs database migrations.** Redeploying code that's already been
tested against the current schema is a fundamentally safer, more automatable
operation than applying a schema change -- those stay a separate, explicit,
manual step (same as every migration this session applied by hand). If a
change needs a migration, apply it yourself first, the same way as before,
then let the pipeline deploy the code that depends on it.

## One-time setup (required before this will work)

The credentials used for the SSM/RDS work in this repo's history have no IAM
permissions -- creating the deploy role needs to be done by you or whoever
manages this AWS account, once:

```bash
./deploy/setup-aws-oidc.sh
```

This creates (idempotently):
- The GitHub OIDC identity provider in IAM, if not already present
- An IAM role (`campusdesk-github-deploy`) that only GitHub Actions runs of
  `ajaypatel01/CampusDesk` can assume (via OIDC -- no long-lived AWS keys
  stored in GitHub at all)
- A policy on that role scoped to exactly `ssm:SendCommand` on the one
  production instance, and `ssm:GetCommandInvocation` (which AWS doesn't
  support scoping to a specific resource)

It prints the resulting role ARN and the exact command to save it as a
repo secret:

```bash
gh secret set AWS_DEPLOY_ROLE_ARN --repo ajaypatel01/CampusDesk --body "<arn>"
```

(or add it manually at Settings -> Secrets and variables -> Actions in the
GitHub web UI, since `gh` isn't authenticated in this environment).

Once that secret exists, the pipeline is live -- the next push to `main`
that touches `frontend/**`, `cmd/**`, or `internal/**` will deploy itself.

## Files

- `backend-deploy.sh` / `frontend-deploy.sh` -- the actual deploy logic,
  runs on the EC2 instance via SSM. Edit these (not the workflow YAML) if
  the deploy steps themselves need to change.
- `run-via-ssm.sh` -- sends a script to the instance via SSM and streams its
  result back into the GitHub Actions log, failing the job if the remote
  script fails.
- `setup-aws-oidc.sh` -- one-time IAM setup, described above.

## Adding a manual approval gate later

The pipeline deploys automatically on push, by design (chosen over a manual
button when this was set up). If you'd rather require a click before
production deploys happen, go to Settings -> Environments -> `production`
in GitHub and add a required reviewer -- no workflow changes needed, since
both jobs already run under the `production` environment.

## Troubleshooting

- Workflow shows the SSM command's full stdout/stderr on failure -- check
  the job log first.
- `AccessDenied` on `sts:AssumeRoleWithWebIdentity`: the trust policy's
  `sub` condition didn't match -- confirm the secret holds the role ARN
  from `setup-aws-oidc.sh`, not a user/access-key ARN.
- Health check fails but the SSM command itself succeeds: that means the
  script's own rollback already ran -- production is back on the previous
  version, but the new code didn't come up healthy. Check the printed
  `journalctl` output in the job log for why.
