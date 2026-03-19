# Marconi — Remaining Work

## Before first real send

- [ ] Get API ID and API Hash from https://my.telegram.org
- [ ] Run `marconi init` or build with embedded credentials:
  ```bash
  go build -ldflags "\
    -X github.com/zhisme/marconi/config.DefaultBotToken=YOUR_TOKEN \
    -X github.com/zhisme/marconi/config.DefaultAPIID=YOUR_ID \
    -X github.com/zhisme/marconi/config.DefaultAPIHash=YOUR_HASH"
  ```
- [ ] Test: `marconi send testdata/formatted.md` — should appear in channel's "Scheduled Messages"
- [ ] Test: `marconi send testdata/formatted.md --now` — sends immediately
- [ ] Test: `marconi preview testdata/formatted.md` — still works (MarkdownV2 output)

## Update marconi-plan.md

The plan doc is now outdated — it still describes Bot API HTTP. Should be updated to reflect the MTProto migration:
- Transport is now gotd/td MTProto, not `net/http` POST to `api.telegram.org`
- Entities are `tg.MessageEntityClass` objects, not `parse_mode: MarkdownV2`
- Session persistence at `~/.config/marconi/session.json`
- Config requires `api_id` + `api_hash` in addition to `bot_token`
- `Sender` interface for testability (no more httptest in send tests)

## CI/CD (GitHub Actions)

- [ ] Update build workflow to pass `-X config.DefaultAPIID` and `-X config.DefaultAPIHash` via secrets
- [ ] Add `MARCONI_API_ID` and `MARCONI_API_HASH` as GitHub repo secrets
- [ ] Binary size will jump ~5MB → ~20MB due to gotd/td — update release notes if needed

## Potential issues to watch for

- **Numeric channel IDs** (`-100xxx`): Currently resolved with `access_hash=0`. If Telegram rejects this, users must use `@username` format instead. The `@username` path uses `ContactsResolveUsername` which always works.
- **Session expiry**: If the session file becomes stale, user gets "invalid bot token or session expired". Fix: delete `~/.config/marconi/session.json` and retry.
- **First-run latency**: MTProto connection + key exchange + bot auth takes a few seconds on first run. Subsequent runs reuse the session and are faster.

## Nice-to-haves (not blocking)

- [ ] `--delete-session` flag to force re-auth
- [ ] Warn if `schedule_date` is <60s in the future or >1 year out (Telegram rejects these)
- [ ] Progress indicator for photo upload (large files)
