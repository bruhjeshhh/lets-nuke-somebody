package game

import (
	"fmt"
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
}

// NewState builds a fresh game state from a set of loaded countries.
// Turn starts at 1 and no country is assigned to the player yet.
func NewState(countries map[string]*Country, order []string) *State {
	return &State{
		Countries: countries,
		Order:     order,
		Turn:      1,
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

// EndTurn advances the game by one turn. Phase 1 has no simulation to run
// yet, so this is currently just a counter increment - the seam where
// economy/AI/combat resolution will hook in later.
func (s *State) EndTurn() int {
	s.Turn++
	return s.Turn
}
