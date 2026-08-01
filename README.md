# Territoria

A turn-based grand strategy game, played in the terminal.

**Phase 1: World Setup** laid the foundation - loading the world, picking
a starting country, and a bare turn counter.

**Phase 2: Economy & Turn Engine** made turns meaningful: every country -
player and AI alike - grows population, earns gold, and recruits troops
each turn, all through one configurable engine.

**Phase 4: Resources & Military Units** (current) expands the economy to
three resources (Gold, Oil, Steel) with per-country production profiles,
and introduces five purchasable unit types (Infantry, Tanks, Artillery,
Fighters, Destroyers) with unit-based combat strength. Actual combat
(attacking a neighbor, territory changing hands) and diplomacy are still
not implemented - this phase gives the engine everything a future combat
system needs (attack/defense totals, resource-gated production) without
yet wiring up an attack command.

## Running it

```bash
go run .
```

By default the game loads `data/countries.json` and `data/units.json`.
Point it at different files with:

```bash
go run . -data path/to/countries.json -units path/to/units.json
```

## Commands

| Command                       | Description                                              |
|--------------------------------|-----------------------------------------------------------|
| `help`                         | List available commands                                   |
| `countries`                    | List every country in the world and its owner              |
| `select <country>`             | Choose your starting country (one-time)                    |
| `stats`                        | Show your country's full status, including military        |
| `resources`                    | Show your resource stockpile and income per turn            |
| `build <unit> <amount>`        | Purchase units (infantry, tanks, artillery, fighters, destroyers) |
| `leaderboard`                  | Show the richest, largest, and strongest countries          |
| `end`                          | Run one turn of economy for every country, then advance     |
| `quit`                         | Exit the game                                               |

## Architecture

The project is split so that game rules never depend on how they're
rendered:

```
internal/game/   Pure game logic: data model, state, economy engine,
                 military/unit system, command execution. No terminal/
                 IO concerns — testable in isolation.

internal/tui/    Bubble Tea model. Only talks to internal/game through
                 its public API (game.Execute, game.State). Owns all
                 layout, styling, and input handling.

main.go          Wiring: load data, build initial state, hand off to
                 the TUI.

data/            JSON world and unit data. countries.json and
                 units.json are the seed datasets; swap in different
                 files via -data / -units for testing.
```

This separation is what will let future phases (combat resolution, AI
behavior, diplomacy) grow inside `internal/game` without ever touching
the rendering layer, and vice versa.

## Economy

Every turn, `internal/game/economy.go` runs the same process for every
country - player and AI, with no special-casing:

1. Population grows by a fixed percentage (`PopulationGrowthRate`).
2. Population and territory are converted into "economic points"
   (`PopulationUnit`, `TerritoryUnit` control the scale).
3. Gold, Oil, and Steel income are each a configurable base rate times
   those points, further scaled by the country's own
   `ResourceMultipliers` - this is what lets an oil producer or an
   industrial/steel power feel different without any per-country logic
   in the engine itself.
4. Troop recruitment (a population-only pool, separate from purchased
   units) uses the same points but ignores resource multipliers.

All of this lives in `EconomyConfig` (see `DefaultEconomyConfig`), so
balancing the game is a config change, not a code change. Adding a
brand new resource later (uranium, food, ...) means: add a rate field
to `EconomyConfig`, add a multiplier field to `ResourceRates` and a
balance field to `Country`, and add one line to `ProjectIncome` -
nothing else in the engine or UI needs to change shape.

## Military units

`internal/game/units.go` (data) and `internal/game/military.go` (logic)
implement purchasable units entirely from `data/units.json` - the
engine has no hardcoded unit names or stats:

- Each unit type has an ID, display name, attack, defense, purchase
  cost, and per-turn maintenance cost (each cost/maintenance figure is
  a `ResourceBundle` of gold/oil/steel).
- `build <unit> <amount>` deducts the full resource cost immediately
  (no partial purchases) and adds the units to the country's inventory.
- `State.MilitaryPower(country)` totals attack, defense, and
  maintenance across every unit a country owns - this is the
  unit-based combat strength the leaderboard and stats panel use in
  place of raw troop count.

Maintenance costs are calculated and displayed (in `stats`) but are
**not yet auto-deducted** each turn - wiring maintenance upkeep into
`EndTurn` is a natural next step once combat gives players a reason to
balance army size against income.

AI countries currently produce resources but don't purchase units -
AI military behavior is out of scope for this phase, same as combat.

## Data model

Each country in `data/countries.json` has:

- `name`, `population`, `territory_size`
- `gold`, `oil`, `steel` (starting resource stockpiles)
- `troops` (population-recruited manpower, separate from units)
- `resource_multipliers` (`gold`/`oil`/`steel` floats; defaults to
  `{1, 1, 1}` if omitted entirely)
- `units` (map of unit ID to count owned; omit for a fresh country
  with none)
- `owner` (`"player"` or `"ai"` — all start as `"ai"` until selected)
- `neighbors` (list of adjacent country names, used by later phases)

Each unit type in `data/units.json` has:

- `id`, `name`
- `attack`, `defense`
- `cost` and `maintenance` (each a `{gold, oil, steel}` bundle)
