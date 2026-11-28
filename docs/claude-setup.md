# Connecting Claude

The cv service writes CVs with Claude through the Claude Code CLI and the owner's Claude subscription, as
`work/reviewer` does. There is no API key. Without a token, the rules engine writes every CV, and the CV page says
so.

The subscription token is for the owner's own use only. Before anyone else uses crashr, the service must move to an
API key (`docs/plans/2026-11-25-cv-service.md`, §1 and §5.4).

## Install the token

1. Create a long-lived token (about a year) with the same CLI version the image pins:

   ```bash
   cd /Users/milicacosmin-sergiu/work/hoply && npx -y @anthropic-ai/claude-code@2.1.322 setup-token
   ```

   It opens the browser to sign in, then prints a token that starts with `sk-ant-oat01-`.

2. Save it where compose mounts it, readable only by you. The file is gitignored:

   ```bash
   cd /Users/milicacosmin-sergiu/work/hoply && (umask 077 && pbpaste > deploy/compose/secrets/claude_oauth_token)
   ```

   Copy the token first. `pbpaste` keeps it out of the shell history.

3. Nothing to restart: the service reads the file on every CV. Check the connection:

   ```bash
   cd /Users/milicacosmin-sergiu/work/hoply && curl -s http://app.localhost/api/cv/v1/writer
   ```

   `"state": "ok"` means connected. `"no_token"` means the file is missing or empty.

## States

| `state` | Meaning | What to do |
| --- | --- | --- |
| `ok` | Claude writes the CVs | nothing |
| `no_token` | no token file | install it (above) |
| `login_needed` | the token was refused: expired or revoked | run step 1 again and replace the file |
| `usage_limit` | the subscription's limit was reached; `limit_until` says until when | wait; the rules write CVs meanwhile |
| `off` | `CV_WRITER` is not `claude-cli` | set it in `docker-compose.yml` |

## Settings (`docker-compose.yml`, service `cv`)

| Variable | Default | Meaning |
| --- | --- | --- |
| `CV_WRITER` | rules | `claude-cli` to write with Claude |
| `CLAUDE_TOKEN_FILE` | | the token file inside the container |
| `CV_MODEL` | `claude-sonnet-5-5` | the model |
| `CV_EFFORT` | `high` | the CLI's `--effort` |
| `CLAUDE_DEADLINE` | `180s` | the longest one run may take |
| `CLAUDE_BIN` | `claude` | the CLI binary |

The CLI version is the `CLAUDE_CODE_VERSION` build argument in `services/cv/Dockerfile`. Bump it on purpose: the
service depends on its flags and result format, which `services/cv/tests/testdata/claude/` records.
