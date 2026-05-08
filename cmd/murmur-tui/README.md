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
- Identity: `g` generate keypair+mnemonic, `1-4` privacy mode
- Waves: `c` compose, `1-8` Wave type, `Enter` submit
- Anonymous: `n` new Specter, `s` switch, `g` gift, `m` mark, `p` mini-games menu
- Onboarding: `Enter` advance phase, `Space` skip onboarding
- Networking: `d` DHT refresh indicator, `r` rate-limit reset indicator

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
