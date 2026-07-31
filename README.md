# Territoria

A turn-based grand strategy game, played in the terminal.

**Phase 1: World Setup** laid the foundation - loading the world, picking
a starting country, and a bare turn counter.

**Phase 2: Economy & Turn Engine** (current) makes turns meaningful:
every country - player and AI alike - grows population, earns gold, and
recruits troops each turn, all through one configurable engine. Combat
and diplomacy are still not implemented; those are later phases built on
top of this scaffolding.

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
| `stats`              | Show detailed info for your country, including income |
| `leaderboard`        | Show the richest and strongest countries in the world |
| `end`                | Run one turn of economy for every country, then advance |
| `quit`               | Exit the game                                      |

## Architecture

The project is split so that game rules never depend on how they're
rendered:

```
internal/game/   Pure game logic: data model, state, economy engine,
                 command execution. No terminal/IO concerns — testable
                 in isolation.

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

## Economy

Every turn, `internal/game/economy.go` runs the same process for every
country - player and AI, with no special-casing:

1. Population grows by a fixed percentage (`PopulationGrowthRate`).
2. Population and territory are converted into "economic points"
   (`PopulationUnit`, `TerritoryUnit` control the scale).
3. Gold income and troop recruitment are each a configurable rate times
   those points.

All of these live in `EconomyConfig` (see `DefaultEconomyConfig`), so
balancing the game is a config change, not a code change.

Adding a new resource later (oil, steel, uranium, ...) means: add a rate
field to `EconomyConfig`, add the matching balance field to `Country`,
and add one line to `Apply()` — nothing else in the engine or UI needs
to change shape.

## Data model

Each country in `data/countries.json` has:

- `name`
- `population`
- `territory_size`
- `gold`
- `troops`
- `owner` (`"player"` or `"ai"` — all start as `"ai"` until selected)
- `neighbors` (list of adjacent country names, used by later phases)
