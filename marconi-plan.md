# Marconi — Go CLI for Telegram Channel Posting

## Context

You want to write formatted posts in Vim and send them to your Telegram channel. The Telegram app doesn't parse markdown on paste, so we need a tool that converts standard Markdown → Telegram MarkdownV2 and sends via the Bot API as a scheduled message with a 24h delay, giving you time to review and edit before it goes live.

Single static binary. No runtime dependencies.

## Names
- **CLI**: `marconi`
- **Telegram bot**: `@marconibot`

## Prerequisites
- Create a Telegram bot via @BotFather (`@MarconiBot`)
- Add the bot to your target channel as **administrator** with permission to post messages
- Configure bot token and channel ID in config file

## Flow
```
marconi send post.md                    # text-only, scheduled 24h from now
marconi send post.md -i photo.jpg       # with image, scheduled 24h from now
marconi send post.md --now              # send immediately (skip delay)
marconi preview post.md                 # dry run, print converted text to terminal
marconi init                            # interactive config setup
```
1. Read markdown file
2. Convert Markdown → MarkdownV2 (via goldmark AST walker)
3. Validate (length limits: 4096 text, 1024 caption)
4. Send to channel via Bot API with `schedule_date` parameter
5. Message appears in channel's "Scheduled Messages" — visible only to admins
6. User reviews/edits directly in Telegram before it auto-publishes
7. `--now` flag skips scheduling and posts immediately

No local queue, no cron, no daemons — Telegram handles the scheduling server-side.

## Why Bot API (not MTProto/TDLib)

The Bot API (`api.telegram.org`) supports everything we need:
- `sendMessage` with `parse_mode: "MarkdownV2"` and `schedule_date`
- `sendPhoto` with caption and `schedule_date`
- Simple HTTP POST — no session management, no auth flow, no C dependencies

MTProto (via TDLib or gotd/td) would be overkill. We don't need user-mode features, real-time updates, or anything beyond "send a message to a channel." Bot API keeps the codebase tiny and the binary dependency-free.

## Project Structure
```
marconi/
├── go.mod
├── go.sum
├── main.go                         # Entry point, CLI routing
├── cmd/
│   ├── send.go                     # send subcommand
│   ├── preview.go                  # preview subcommand
│   └── init.go                     # init subcommand (interactive config setup)
├── converter/
│   ├── converter.go                # Markdown → MarkdownV2 via goldmark AST
│   ├── renderer.go                 # goldmark ast.Walker that emits MarkdownV2
│   ├── escape.go                   # MarkdownV2 escaping helpers
│   └── converter_test.go           # Heaviest test file
├── telegram/
│   ├── client.go                   # Bot API HTTP client (sendMessage, sendPhoto)
│   └── client_test.go
├── config/
│   ├── config.go                   # YAML config loader
│   └── config_test.go
├── validator/
│   ├── validator.go                # Length checks
│   └── validator_test.go
└── testdata/
    └── *.md                        # Test markdown fixtures
```

## Key Dependencies
- **goldmark** (`github.com/yuin/goldmark`) — Markdown parser with full AST access. We walk the AST and emit MarkdownV2 directly — same two-layer escaping approach as before, just using tree traversal instead of callbacks.
- **gopkg.in/yaml.v3** — YAML config parsing
- **net/http** (stdlib) — Bot API calls. No third-party Telegram library needed — the Bot API is just HTTP POST with JSON/multipart.

Zero C dependencies. Zero system-level installs. `go build` → done.

## Core Design: Markdown → MarkdownV2 Converter

Walk the goldmark AST and emit MarkdownV2 string. The AST walker visits nodes on Enter and Exit:

### Node type mapping
- **ast.Text** → escape all MarkdownV2 special chars
- **ast.Emphasis (level 1)** → wrap with `_..._` (italic)
- **ast.Emphasis (level 2)** → wrap with `*...*` (bold)
- **ast.Strikethrough** (extension) → wrap with `~...~`
- **ast.CodeSpan** → wrap with `` `...` `` (only escape `` ` `` and `\` inside)
- **ast.FencedCodeBlock** → wrap with ` ```lang\n...\n``` `
- **ast.Link** → `[content](url)` (escape `)` and `\` in URL)
- **ast.Blockquote** → prefix lines with `>`
- **ast.Heading** → render as `*text*` (bold — no headers in Telegram)
- **ast.List / ast.ListItem** → handle ordered (numbered) and unordered (bullet) lists

### Escaping Strategy

Same insight as before, mapped to AST traversal:

- **Text nodes** — escape all MarkdownV2 special chars. This is the only place full escaping happens.
- **Emphasis, Strikethrough, etc.** — on Enter, emit opening delimiter; on Exit, emit closing delimiter. Inner content is already escaped by child text nodes.
- **CodeSpan / FencedCodeBlock** — only escape `` ` `` and `\` inside. Raw text, no child node escaping.
- **Link** — content children handle their own escaping; URL portion escapes only `)` and `\`.

Two-layer approach preserved: escape in text nodes, wrap in structural nodes.

### Special Characters to Escape
```
_ * [ ] ( ) ~ ` > # + - = | { } . ! \
```

### goldmark Setup
```go
md := goldmark.New(
    goldmark.WithExtensions(
        extension.Strikethrough,
    ),
    goldmark.WithParserOptions(
        parser.WithAutoHeadingID(),
    ),
)
// Parse to AST, then walk with our custom renderer
doc := md.Parser().Parse(text.NewReader(source))
```

## Config (`~/.config/marconi/config.yml`)
```yaml
bot_token: "123456:ABC-DEF..."   # MarconiBot token from @BotFather
channel_id: "@my_channel"        # Channel username or numeric ID
delay_hours: 24                  # Default scheduling delay (0 = immediate)
```
Config source: `~/.config/marconi/config.yml` (file) or CLI inline flags (`--token`, `--channel`). CLI flags override config file values.

No `api_id`/`api_hash` needed — Bot API only requires the bot token. Config is simpler.

## Telegram Bot API

Direct HTTP calls to `https://api.telegram.org/bot<token>/`. No SDK wrapper needed.

### Text post (scheduled)
```go
params := url.Values{
    "chat_id":    {cfg.ChannelID},
    "text":       {convertedText},
    "parse_mode": {"MarkdownV2"},
}
if !now {
    params.Set("schedule_date", strconv.FormatInt(time.Now().Unix()+int64(cfg.DelayHours*3600), 10))
}
resp, err := http.PostForm("https://api.telegram.org/bot"+cfg.BotToken+"/sendMessage", params)
```

### Photo + caption (scheduled)
```go
// multipart/form-data with photo file + caption fields
body := &bytes.Buffer{}
writer := multipart.NewWriter(body)
writer.WriteField("chat_id", cfg.ChannelID)
writer.WriteField("caption", convertedText)
writer.WriteField("parse_mode", "MarkdownV2")
if !now {
    writer.WriteField("schedule_date", strconv.FormatInt(time.Now().Unix()+int64(cfg.DelayHours*3600), 10))
}
part, _ := writer.CreateFormFile("photo", filepath.Base(imagePath))
io.Copy(part, photoFile)
writer.Close()

resp, err := http.Post("https://api.telegram.org/bot"+cfg.BotToken+"/sendPhoto", writer.FormDataContentType(), body)
```

Bot must be channel admin with "Post Messages" permission.

## Distribution

### Pre-built binaries (recommended)

Download from [GitHub Releases](https://github.com/zhisme/marconi/releases). Binaries are available for:
- **Linux** — amd64, arm64
- **macOS** — amd64 (Intel), arm64 (Apple Silicon)
- **Windows** — amd64

Pre-built binaries ship with `@MarconiPostBot` token embedded. Users only need to provide their channel ID.

### Build from source

If your platform isn't covered or you prefer to build yourself, create your own bot via `@BotFather` and build with your token:

```bash
git clone https://github.com/zhisme/marconi.git && cd marconi
go build -ldflags "-X github.com/zhisme/marconi/config.DefaultBotToken=YOUR_BOT_TOKEN" -o marconi .
```

### CI/CD

GitHub Actions builds and publishes release binaries on git tag push. The bot token is stored as a GitHub repository secret (`MARCONI_BOT_TOKEN`) and injected at build time via `-ldflags`. It never appears in source code.

## Execution Flow

### `marconi send post.md -i photo.jpg`

```
┌─────────────────────────────────────────────────────────────────────┐
│ main.go                                                             │
│                                                                     │
│  os.Args → route to cmd/send.go                                    │
│  parse flags: file="post.md", image="photo.jpg", now=false         │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│ config.Load(cliFlags)                                               │
│                                                                     │
│  1. Load ~/.config/marconi/config.yml (if exists)                   │
│  2. Override with CLI flags (--token, --channel, --delay)           │
│  3. Parse YAML → Config{BotToken, ChannelID, DelayHours}           │
│  4. Return error if required fields missing → "run marconi init"   │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│ os.ReadFile("post.md")                                              │
│                                                                     │
│  Read raw markdown bytes from disk                                  │
│  Return error if file doesn't exist                                 │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│ converter.Convert(source []byte) → (string, error)                  │
│                                                                     │
│  1. goldmark parser creates AST from source bytes                   │
│     md.Parser().Parse(text.NewReader(source)) → ast.Document        │
│                                                                     │
│  2. Walk AST with custom TelegramRenderer                           │
│     ast.Walk(doc, func(node, entering) { ... })                     │
│                                                                     │
│  3. For each node, the walker does:                                 │
│                                                                     │
│     ast.Text (leaf):                                                │
│       → escapeMarkdownV2("Hello world!") → "Hello world\!"         │
│       → write to strings.Builder                                    │
│                                                                     │
│     ast.Emphasis, level=2 (entering=true):                          │
│       → write "*"                                                   │
│     ast.Emphasis, level=2 (entering=false):                         │
│       → write "*"                                                   │
│       Result: *already\-escaped inner text*                         │
│                                                                     │
│     ast.CodeSpan (entering=true):                                   │
│       → read raw code segment bytes from source                     │
│       → escapeCodeSpan(raw) (only escape ` and \)                   │
│       → write "`escaped`"                                           │
│       → return ast.WalkSkipChildren (don't visit children)          │
│                                                                     │
│     ast.FencedCodeBlock (entering=true):                            │
│       → read language info → "go"                                   │
│       → read raw code lines from source                             │
│       → escapeCodeBlock(raw) (only escape ` and \)                  │
│       → write "```go\nescaped code\n```"                            │
│       → return ast.WalkSkipChildren                                 │
│                                                                     │
│     ast.Link (entering=true):                                       │
│       → write "["                                                   │
│     ast.Link (entering=false):                                      │
│       → url := escapeURL(node.Destination) (escape ) and \)         │
│       → write "](url)"                                              │
│                                                                     │
│     ast.Blockquote (entering=true):                                 │
│       → set flag: insideBlockquote = true                           │
│     ast.Blockquote (entering=false):                                │
│       → clear flag                                                  │
│     (Text nodes check this flag and prefix lines with >)            │
│                                                                     │
│     ast.Heading (entering=true):                                    │
│       → write "*" (render as bold)                                  │
│     ast.Heading (entering=false):                                   │
│       → write "*\n"                                                 │
│                                                                     │
│     ast.List / ast.ListItem:                                        │
│       → track ordered vs unordered, maintain counter                │
│       → prefix with "1\. " or "• "                                  │
│                                                                     │
│     ast.Paragraph:                                                  │
│       → entering: noop                                              │
│       → exiting: write "\n\n" (unless inside list item)             │
│                                                                     │
│  4. Return builder.String()                                         │
│                                                                     │
│  Example:                                                           │
│    Input:  "Hello **world**, check `code`!"                         │
│    AST:    Paragraph > [Text, Emphasis>Text, Text, CodeSpan, Text]  │
│    Output: "Hello *world*, check `code`\!"                          │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│ validator.Validate(text string, hasImage bool) → error              │
│                                                                     │
│  if hasImage && len(text) > 1024:                                   │
│    → error: "caption too long (N/1024 chars)"                       │
│  if !hasImage && len(text) > 4096:                                  │
│    → error: "message too long (N/4096 chars)"                       │
│                                                                     │
│  Note: validates converted MarkdownV2 length, not source markdown.  │
│  Escaping inflates length (every special char gets a \ prefix).     │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│ telegram.Client.Send(text, imagePath, now) → error                  │
│                                                                     │
│  Compute schedule_date:                                             │
│    if !now → scheduleDate = time.Now().Unix() + delayHours*3600     │
│                                                                     │
│  Branch on imagePath:                                               │
│                                                                     │
│  ┌─── imagePath == "" (text only) ──────────────────────────────┐   │
│  │                                                               │   │
│  │  POST https://api.telegram.org/bot<token>/sendMessage         │   │
│  │  Content-Type: application/x-www-form-urlencoded              │   │
│  │                                                               │   │
│  │  chat_id=@my_channel                                          │   │
│  │  text=Hello+*world*,+check+`code`\!                           │   │
│  │  parse_mode=MarkdownV2                                        │   │
│  │  schedule_date=1742313600          ← (unix timestamp, 24h)    │   │
│  │                                                               │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─── imagePath != "" (photo + caption) ────────────────────────┐   │
│  │                                                               │   │
│  │  POST https://api.telegram.org/bot<token>/sendPhoto           │   │
│  │  Content-Type: multipart/form-data; boundary=---xyz           │   │
│  │                                                               │   │
│  │  ---xyz                                                       │   │
│  │  Content-Disposition: form-data; name="chat_id"               │   │
│  │  @my_channel                                                  │   │
│  │  ---xyz                                                       │   │
│  │  Content-Disposition: form-data; name="caption"               │   │
│  │  Hello *world*, check `code`\!                                │   │
│  │  ---xyz                                                       │   │
│  │  Content-Disposition: form-data; name="parse_mode"            │   │
│  │  MarkdownV2                                                   │   │
│  │  ---xyz                                                       │   │
│  │  Content-Disposition: form-data; name="schedule_date"         │   │
│  │  1742313600                                                   │   │
│  │  ---xyz                                                       │   │
│  │  Content-Disposition: form-data; name="photo";                │   │
│  │    filename="photo.jpg"                                       │   │
│  │  Content-Type: image/jpeg                                     │   │
│  │  <binary data>                                                │   │
│  │  ---xyz--                                                     │   │
│  │                                                               │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  Parse response JSON:                                               │
│    { "ok": true, "result": { "message_id": 42, ... } }             │
│    → success: print "Scheduled for <time>" or "Sent!"              │
│                                                                     │
│    { "ok": false, "description": "Bad Request: ..." }              │
│    → error: print Telegram's error description                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### `marconi preview post.md`

Same as `send` but stops after converter — prints MarkdownV2 to stdout, no API call.

```
main.go → config.Load() → os.ReadFile() → converter.Convert() → fmt.Println(result)
```

### `marconi init`

Interactive config setup, no conversion or API calls.

```
main.go → cmd/init.go
  → prompt: "Bot token: "       → read stdin
  → prompt: "Channel ID: "      → read stdin
  → prompt: "Delay hours [24]: " → read stdin (default 24)
  → write YAML to ~/.config/marconi/config.yml
  → print "Config saved to ~/.config/marconi/config.yml"
```

### Error flow

Errors bail early with a clear message. No stack traces for user-facing errors.

```
config missing/incomplete → "Missing bot_token/channel_id. Run: marconi init"
file not found          → "File not found: post.md"
converter parse error   → "Failed to convert markdown: <detail>"
validation failure      → "Caption too long: 1200/1024 chars (source: 980 chars before escaping)"
API 400                 → "Telegram rejected the message: <their error string>"
API 401                 → "Invalid bot token. Run: marconi init"
API 403                 → "Bot is not an admin in this channel"
network error           → "Cannot reach Telegram API: <detail>"
```

## Implementation Phases
1. **Skeleton** — `go mod init`, main.go with subcommand routing, `marconi --help` works
2. **Config** — config package + `marconi init` (bot_token, channel_id, delay_hours)
3. **Converter** — goldmark AST walker + MarkdownV2 renderer + extensive tests (longest phase)
4. **Telegram Client** — Bot API HTTP client: sendMessage, sendPhoto with schedule_date
5. **Validator** — Length checks (4096 text / 1024 caption)
6. **Workflow** — Wire everything in send command: read → convert → validate → send
7. **Polish** — Error messages, edge cases

## Known Challenges
- Ordered list numbering: track counter in walker state per list
- Escaping inflates text length: validate on converted text, report both lengths
- goldmark's `no_intra_emphasis` equivalent: not needed — goldmark handles `snake_case` correctly by default
- Unicode/emoji pass through unescaped (only ASCII special chars need escaping)
- Nested blockquotes: prefix each line with appropriate number of `>` chars
- Bot API `schedule_date` requires the date to be at least 60 seconds in the future and no more than ~1 year

## Verification
1. `go test ./...` — all tests pass
2. `marconi init` — creates config interactively
3. `marconi preview test.md` — prints converted MarkdownV2 to stdout
4. `marconi send test.md` — sends scheduled message to channel (publishes in 24h)
5. `marconi send test.md --now` — sends immediately to channel
6. `marconi send test.md -i photo.jpg` — sends scheduled message with image
7. Verify in Telegram: message visible in channel's "Scheduled Messages" section
