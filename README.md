## Let's Nuke Somebody

A turn-based grand strategy game, played in the terminal.

This is **Phase 1: World Setup** — the project foundation. It covers
loading the world, picking a starting country, and a bare turn counter.
No combat, economy, diplomacy, or AI behavior yet; those are later phases
built on top of this scaffolding.

## Running it

```bash
go run .
```

By default the game loads `data/countries.json`. Point it at a different
file with:

```bash
go run . -data path/to/countries.json
```

## Commands

| Command             | Description                                       |
|----------------------|----------------------------------------------------|
| `help`               | List available commands                            |
| `countries`          | List every country in the world and its owner      |
| `select <country>`   | Choose your starting country (one-time)            |
| `stats`              | Show detailed info for your country                |
| `end`                | End the current turn and advance to the next       |
| `quit`               | Exit the game                                      |

## Architecture

The project is split so that game rules never depend on how they're
rendered:

```
internal/game/   Pure game logic: data model, state, command execution.
                 No terminal/IO concerns — testable in isolation.

internal/tui/    Bubble Tea model. Only talks to internal/game through
                 its public API (game.Execute, game.State). Owns all
                 layout, styling, and input handling.

main.go          Wiring: load data, build initial state, hand off to
                 the TUI.

data/            JSON world data. countries.json is the seed dataset;
                 swap in a different file via -data for testing.
```

This separation is what will let future phases (combat, economy, AI
behavior, diplomacy) grow inside `internal/game` without ever touching
the rendering layer, and vice versa.

## Data model

Each country in `data/countries.json` has:

- `name`
- `population`
- `territory_size`
- `gold`
- `troops`
- `owner` (`"player"` or `"ai"` — all start as `"ai"` until selected)
- `neighbors` (list of adjacent country names, used by later phases)
