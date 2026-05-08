# TUI Architecture

`cmd/murmur-tui` uses Bubble Tea Elm architecture with a single root model and composable sub-models.

## Model hierarchy

- `pkg/tui/model.go` — root `Model`
  - Pulse Map view model (`pkg/tui/views/pulsemap.go`)
  - Identity view model (`pkg/tui/views/identity.go`)
  - Waves view model (`pkg/tui/views/waves.go`)
  - Anonymous view model (`pkg/tui/views/anonymous.go`)
  - Onboarding view model (`pkg/tui/views/onboarding.go`)
  - Networking view model (`pkg/tui/views/networking.go`)

Shared state:
- `pkg/tui/views/state.go` holds keypair/mode/specter state used across views.

## Input and styling

- `pkg/tui/input/keymap.go` centralizes key bindings.
- `pkg/tui/styles/theme.go` centralizes Lipgloss style tokens.
- `pkg/tui/components/*` provides reusable render components (tabs/status/help).

## Event bridge

- `pkg/tui/bridge/eventbus.go` provides an event-stream abstraction that feeds external events into `tea.Msg`.
- Root model consumes bridge messages and applies them to networking/UI state.

## Update/View flow

1. Root receives `tea.Msg`
2. Global keys handled first (quit/help/tab routing)
3. Message routed to active sub-model
4. Sub-model returns updated state and optional `tea.Cmd`
5. Root renders tabs + active panel + status/help overlays

