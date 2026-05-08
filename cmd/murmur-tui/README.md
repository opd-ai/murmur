# murmur-tui

Bubble Tea terminal UI for MURMUR.

## Build

```bash
make build
```

Produces:
- `bin/murmur`
- `bin/murmur-tui`

Or build only TUI:

```bash
go build ./cmd/murmur-tui
```

## Run

```bash
./bin/murmur-tui
```

## Keybindings

- Global: `q`/`Ctrl+C` quit, `?` help, `Tab`/`Shift+Tab` cycle views, `1-6` jump views
- Pulse Map: `h/j/k/l` or arrows pan, `+/-` zoom, `n` fit/reset, `Enter` select
- Pulse Map: `/` search, `m` actions menu, `Ctrl+B` bookmark, `Ctrl+1..9` bookmark jump
- Pulse Map: `z` center/zoom focus (double-tap equivalent), color/state tags, overlay glyph annotations
- Identity: `g` generate keypair+mnemonic, `1-4` privacy mode, `d` declaration mode, `u` publish declaration
- Waves: `c` compose, `1-8` Wave type, `a` amplify, `y` reply mode, `Enter` submit
- Anonymous: `n` new Specter, `s` switch, `c` circuit, `r` relays, `w` whisper, `p` mini-games menu (`1-6` boards)
- Onboarding: `Enter` advance, `Space` skip/resume marker, `i` invitation warm-start, `r` recovery branch, `x/a` hint dismiss/ack
- Networking: `d` DHT refresh indicator, `r` rate-limit reset indicator, `n` relay diagnostics state, live peer feed
- Settings: `Ctrl+,` toggle settings modal; numeric controls for mode/blend/overlays; `r/o/i/h` config toggles

## Mouse

- Click: select/activate
- Drag: Pulse Map pan
- Wheel: zoom/scroll

## ASCII snapshots

### Pulse Map

```text
MURMUR TUI
[Pulse Map] [Identity] [Waves] ...
Camera x=0.0 y=0.0 zoom=1.00
Nodes:
▶ ● self     (  0.0,  0.0)
  ◉ peer-a   (  6.0, -2.0)
```

### Identity

```text
Mode: Open
Public key: 9f6c... 
Mnemonic: word1 word2 ...
Sigil:
▓▒░█...
```
