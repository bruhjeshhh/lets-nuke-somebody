# Territoria

A turn-based grand strategy game, played in the terminal.

**Phase 1: World Setup** laid the foundation - loading the world, picking
a starting country, and a bare turn counter.

**Phase 2: Economy & Turn Engine** made turns meaningful: every country -
player and AI alike - grows population, earns gold, and recruits troops
each turn, all through one configurable engine.

**Phase 3: Military & Combat** adds troop recruitment and conquest:
attack a neighboring country, resolve the battle by troop count, and
capture its territory on a win. AI countries recruit automatically;
they don't attack yet.

**Phase 4: Resources & Military Units** (current) expands the economy to
three resources (Gold, Oil, Steel) with per-country production profiles,
and introduces five purchasable unit types (Infantry, Tanks, Artillery,
Fighters, Destroyers) with a unit-based strength calculation
(`State.MilitaryPower`). Combat resolution itself still runs on troop
count only (see Combat below for why, and how the two connect) -
diplomacy remains unimplemented.

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
| `recruit <amount>`             | Recruit troops with gold                                   |
| `attack <country>`             | Attack a neighboring country                                |
| `map`                          | Show world ownership - your empire vs. everyone else        |
| `leaderboard`                  | Show the richest, largest, and strongest countries          |
| `end`                          | Run one turn of economy for every country, then advance     |
| `quit`                         | Exit the game                                               |

## Architecture

The project is split so that game rules never depend on how they're
rendered:

```
internal/game/   Pure game logic: data model, state, economy engine,
                 military/unit system, combat/conquest, command
                 execution. No terminal/IO concerns — testable in
                 isolation.

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
`EndTurn` is a natural next step once army size becomes something
players have to balance against income.

AI countries produce resources and auto-recruit troops (see Combat
below) but don't purchase units yet - AI unit-buying behavior is a
natural extension once combat also considers unit strength.

## Combat

`internal/game/combat.go` implements recruitment and conquest:

- `recruit <amount>` buys troops for the player's country at a
  configurable gold-per-troop rate (`CombatConfig.GoldPerTroop`). AI
  countries recruit automatically each turn, spending a configurable
  fraction of their gold (`AIRecruitGoldFraction`) the same way - see
  `State.aiAutoRecruit`, called from `EndTurn`.
- `attack <country>` can only target a country adjacent to the
  player's - `Only neighboring countries can be attacked` is enforced
  by checking the attacker's `Neighbors` list.
- Combat strength is **troop count only** (`CombatStrength`), per this
  phase's spec. It's deliberately factored into its own one-line
  function rather than inlined into `Attack`, specifically so folding
  in the unit-based `State.MilitaryPower` from Phase 4 later is a
  change to `CombatStrength` alone, not to `Attack`'s control flow.
- Both sides take casualties every battle (`WinnerCasualtyRate`,
  `LoserCasualtyRate`) regardless of outcome - the winner just loses
  less. Ties favor the defender.
- A winning attacker gains the defender's territory (`Owner` flips to
  `player`) and absorbs its neighbor connections, so conquest chains
  outward through captured ground on future attacks rather than
  dead-ending at your original borders.
- There's no province/tile subdivision in this data model - each
  country is one indivisible territory, so a single battle either
  fully captures a country or doesn't touch it at all. That's the
  "Simple Version" the spec asked for; a granular front-line/partial-
  territory model would be a larger data model change for a later
  phase.
- The player's home country (`State.PlayerCountry`) is the sole
  attacker/troop pool for now - conquered countries join "countries
  you control" (see `stats`/`map`) but don't independently recruit or
  launch attacks yet.

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
