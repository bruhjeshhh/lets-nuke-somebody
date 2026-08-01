package game

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ResourceBundle is a set of resource amounts - used both as a unit's
// purchase cost / per-turn maintenance, and as a scratch value when
// totaling costs across a purchase of several units at once.
type ResourceBundle struct {
	Gold  int `json:"gold"`
	Oil   int `json:"oil"`
	Steel int `json:"steel"`
}

// Scale multiplies every field by n - used to turn a single unit's cost
// into the cost of buying n of them.
func (r ResourceBundle) Scale(n int) ResourceBundle {
	return ResourceBundle{Gold: r.Gold * n, Oil: r.Oil * n, Steel: r.Steel * n}
}

// Add combines two bundles - used to total maintenance across a whole
// military.
func (r ResourceBundle) Add(o ResourceBundle) ResourceBundle {
	return ResourceBundle{Gold: r.Gold + o.Gold, Oil: r.Oil + o.Oil, Steel: r.Steel + o.Steel}
}

// IsZero reports whether every field is zero, e.g. a unit with no
// maintenance cost at all.
func (r ResourceBundle) IsZero() bool {
	return r.Gold == 0 && r.Oil == 0 && r.Steel == 0
}

// UnitType is one purchasable military unit definition - Infantry,
// Tanks, Artillery, Fighters, Destroyers, or whatever is added later
// purely by editing data/units.json. Combat strength and purchase
// rules are entirely data-driven: nothing in the engine hardcodes unit
// names or stats, so rebalancing or adding a new unit type never
// requires a code change.
type UnitType struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Attack      int            `json:"attack"`
	Defense     int            `json:"defense"`
	Cost        ResourceBundle `json:"cost"`
	Maintenance ResourceBundle `json:"maintenance"`
}

// LoadUnitTypes reads a JSON array of unit definitions from path and
// returns them as a lookup map keyed by ID, plus a slice preserving
// source file order for stable display.
func LoadUnitTypes(path string) (map[string]*UnitType, []string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading unit data: %w", err)
	}

	var list []UnitType
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, nil, fmt.Errorf("parsing unit data: %w", err)
	}
	if len(list) == 0 {
		return nil, nil, fmt.Errorf("no unit types found in %s", path)
	}

	units := make(map[string]*UnitType, len(list))
	order := make([]string, 0, len(list))

	for i := range list {
		u := list[i]
		if u.ID == "" {
			return nil, nil, fmt.Errorf("unit type at index %d is missing an id", i)
		}
		if _, exists := units[u.ID]; exists {
			return nil, nil, fmt.Errorf("duplicate unit type id: %s", u.ID)
		}
		unitCopy := u
		units[u.ID] = &unitCopy
		order = append(order, u.ID)
	}

	return units, order, nil
}

// FindUnitType resolves a unit type by ID or display name,
// case-insensitively, so command input like "build Tanks 5" or
// "build TANKS 5" both work.
func (s *State) FindUnitType(name string) (*UnitType, bool) {
	if u, ok := s.Units[name]; ok {
		return u, true
	}
	target := strings.ToLower(strings.TrimSpace(name))
	for _, id := range s.UnitOrder {
		u := s.Units[id]
		if strings.ToLower(u.ID) == target || strings.ToLower(u.Name) == target {
			return u, true
		}
	}
	return nil, false
}

// MilitaryPower totals a country's combat strength and upkeep from its
// owned units: total attack, total defense, and total maintenance
// across every resource. This is the "combat uses unit statistics
// instead of only troop count" calculation - it doesn't run a battle
// by itself, but it's the number any future combat/invasion command
// would compare between attacker and defender.
func (s *State) MilitaryPower(c *Country) (attack int, defense int, maintenance ResourceBundle) {
	for _, id := range s.UnitOrder {
		count := c.Units[id]
		if count == 0 {
			continue
		}
		unit := s.Units[id]
		attack += count * unit.Attack
		defense += count * unit.Defense
		maintenance = maintenance.Add(unit.Maintenance.Scale(count))
	}
	return
}

// BuildUnits purchases `amount` of the given unit type for the named
// country, deducting the total resource cost immediately. It fails
// (with no side effects) if the unit type or country doesn't exist, the
// amount isn't positive, or the country can't afford the full cost -
// there's no partial purchase.
func (s *State) BuildUnits(countryName, unitName string, amount int) (*UnitType, ResourceBundle, error) {
	if amount <= 0 {
		return nil, ResourceBundle{}, fmt.Errorf("amount must be a positive number")
	}

	c, ok := s.FindCountry(countryName)
	if !ok {
		return nil, ResourceBundle{}, fmt.Errorf("no country named %q", countryName)
	}

	unit, ok := s.FindUnitType(unitName)
	if !ok {
		return nil, ResourceBundle{}, fmt.Errorf("no unit type named %q", unitName)
	}

	cost := unit.Cost.Scale(amount)

	var shortfalls []string
	if c.Gold < cost.Gold {
		shortfalls = append(shortfalls, fmt.Sprintf("gold (need %d, have %d)", cost.Gold, c.Gold))
	}
	if c.Oil < cost.Oil {
		shortfalls = append(shortfalls, fmt.Sprintf("oil (need %d, have %d)", cost.Oil, c.Oil))
	}
	if c.Steel < cost.Steel {
		shortfalls = append(shortfalls, fmt.Sprintf("steel (need %d, have %d)", cost.Steel, c.Steel))
	}
	if len(shortfalls) > 0 {
		return nil, ResourceBundle{}, fmt.Errorf("not enough resources: %s", strings.Join(shortfalls, ", "))
	}

	c.Gold -= cost.Gold
	c.Oil -= cost.Oil
	c.Steel -= cost.Steel
	if c.Units == nil {
		c.Units = map[string]int{}
	}
	c.Units[unit.ID] += amount

	return unit, cost, nil
}
