package game

import (
	"fmt"
	"sort"
	"strings"
)

// State holds everything about an in-progress game: the world of
// countries, whose turn it is, and which country (if any) the human
// player controls.
//
// State intentionally contains no rendering logic - it is safe to unit
// test or drive from a non-interactive caller.
type State struct {
	Countries     map[string]*Country
	Order         []string // stable display order, matches source data file
	PlayerCountry string   // empty until the player has selected one
	Turn          int
	Economy       EconomyConfig
	Combat        CombatConfig

	// Units is the catalog of purchasable unit types (loaded once from
	// units.json), keyed by ID. UnitOrder preserves source file order
	// for stable display, same pattern as Countries/Order.
	Units     map[string]*UnitType
	UnitOrder []string
}

// NewState builds a fresh game state from a set of loaded countries and
// a catalog of unit types. Turn starts at 1, no country is assigned to
// the player yet, and the economy/combat balance are set to their
// defaults (but freely tunable).
func NewState(countries map[string]*Country, order []string, units map[string]*UnitType, unitOrder []string) *State {
	return &State{
		Countries: countries,
		Order:     order,
		Turn:      1,
		Economy:   DefaultEconomyConfig(),
		Combat:    DefaultCombatConfig(),
		Units:     units,
		UnitOrder: unitOrder,
	}
}

// FindCountry resolves a country by name, case-insensitively, so command
// input like "select united states" still works.
func (s *State) FindCountry(name string) (*Country, bool) {
	if c, ok := s.Countries[name]; ok {
		return c, true
	}
	target := strings.ToLower(strings.TrimSpace(name))
	for _, n := range s.Order {
		if strings.ToLower(n) == target {
			return s.Countries[n], true
		}
	}
	return nil, false
}

// SelectCountry assigns the given country to the player. It fails if the
// player has already selected a country, or if the country does not
// exist. All other countries remain (or become) AI-controlled.
func (s *State) SelectCountry(name string) (*Country, error) {
	if s.PlayerCountry != "" {
		return nil, fmt.Errorf("you already control %s; starting country cannot be changed", s.PlayerCountry)
	}

	c, ok := s.FindCountry(name)
	if !ok {
		return nil, fmt.Errorf("no country named %q", name)
	}

	c.Owner = OwnerPlayer
	s.PlayerCountry = c.Name

	for _, n := range s.Order {
		if n == c.Name {
			continue
		}
		s.Countries[n].Owner = OwnerAI
	}

	return c, nil
}

// PlayerHasSelected reports whether the player has chosen a starting
// country yet.
func (s *State) PlayerHasSelected() bool {
	return s.PlayerCountry != ""
}

// TurnSummary reports what happened on a completed turn, focused on the
// player's country (AI countries run through the identical economy but
// aren't individually reported on here - see Leaderboard for the
// world-wide view).
type TurnSummary struct {
	Turn          int
	PlayerCountry string // empty if the player hasn't selected a country yet
	Result        TurnResult
}

// EndTurn advances the game by one turn and runs one round of economy
// (population growth, gold income, troop recruitment) for every country
// in the world - player and AI alike, via the same Apply function, so
// there's no special-casing between them. AI countries then spend a
// share of their income automatically recruiting troops; the player
// recruits manually via the 'recruit' command instead.
func (s *State) EndTurn() TurnSummary {
	s.Turn++

	summary := TurnSummary{Turn: s.Turn}
	for _, name := range s.Order {
		c := s.Countries[name]
		result := Apply(c, s.Economy)
		if name == s.PlayerCountry {
			summary.PlayerCountry = name
			summary.Result = result
		}
	}
	s.aiAutoRecruit()
	return summary
}

// RankedByGold returns every country ordered richest-to-poorest.
func (s *State) RankedByGold() []*Country {
	return s.ranked(func(c *Country) int64 { return int64(c.Gold) })
}

// RankedByTroops returns every country ordered strongest-to-weakest by
// troop count.
func (s *State) RankedByTroops() []*Country {
	return s.ranked(func(c *Country) int64 { return int64(c.Troops) })
}

// RankedByMilitaryPower returns every country ordered by total military
// strength (attack + defense across all owned units) - this is the
// unit-based combat metric, distinct from raw troop count.
func (s *State) RankedByMilitaryPower() []*Country {
	return s.ranked(func(c *Country) int64 {
		attack, defense, _ := s.MilitaryPower(c)
		return int64(attack + defense)
	})
}

// ranked returns all countries sorted descending by the given metric,
// breaking ties by name so ordering is deterministic.
func (s *State) ranked(metric func(*Country) int64) []*Country {
	countries := make([]*Country, 0, len(s.Order))
	for _, name := range s.Order {
		countries = append(countries, s.Countries[name])
	}
	sort.Slice(countries, func(i, j int) bool {
		mi, mj := metric(countries[i]), metric(countries[j])
		if mi != mj {
			return mi > mj
		}
		return countries[i].Name < countries[j].Name
	})
	return countries
}
