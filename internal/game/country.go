package game

import (
	"encoding/json"
	"fmt"
	"os"
)

// Owner identifies who controls a country.
type Owner string

const (
	OwnerPlayer Owner = "player"
	OwnerAI     Owner = "ai"
)

// Country represents a single nation in the game world.
//
// Neighbors is stored as a list of country names rather than pointers so
// that the struct can be marshaled/unmarshaled directly from JSON. Callers
// that need the actual neighboring Country objects should resolve names
// through a World/State lookup.
type Country struct {
	Name          string   `json:"name"`
	Population    int64    `json:"population"`
	TerritorySize int      `json:"territory_size"`
	Gold          int      `json:"gold"`
	Oil           int      `json:"oil"`
	Steel         int      `json:"steel"`
	Troops        int      `json:"troops"`
	Owner         Owner    `json:"owner"`
	Neighbors     []string `json:"neighbors"`

	// ResourceMultipliers scales the base economy formulas per resource,
	// so a country's real-world strengths (e.g. an oil producer, an
	// industrial/steel power) are reflected without hardcoding any
	// per-country logic into the economy engine itself. A zero-value
	// (all three fields unset in the source JSON) is treated as an
	// even {1, 1, 1} baseline - see LoadCountries.
	ResourceMultipliers ResourceRates `json:"resource_multipliers"`

	// Units maps a unit type ID (matching UnitType.ID in units.json,
	// e.g. "tanks") to how many of that unit this country currently
	// owns. Missing entries mean zero, so most countries can omit this
	// entirely in the source JSON.
	Units map[string]int `json:"units,omitempty"`
}

// LoadCountries reads a JSON array of countries from path and returns them
// as a lookup map keyed by country name, along with a slice preserving the
// original file order (JSON map iteration order in Go is randomized, so we
// keep this separately for stable display ordering).
func LoadCountries(path string) (map[string]*Country, []string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading country data: %w", err)
	}

	var list []Country
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, nil, fmt.Errorf("parsing country data: %w", err)
	}
	if len(list) == 0 {
		return nil, nil, fmt.Errorf("no countries found in %s", path)
	}

	countries := make(map[string]*Country, len(list))
	order := make([]string, 0, len(list))

	for i := range list {
		c := list[i]
		if c.Name == "" {
			return nil, nil, fmt.Errorf("country at index %d is missing a name", i)
		}
		if _, exists := countries[c.Name]; exists {
			return nil, nil, fmt.Errorf("duplicate country name: %s", c.Name)
		}
		if c.Owner == "" {
			c.Owner = OwnerAI
		}
		if c.ResourceMultipliers == (ResourceRates{}) {
			c.ResourceMultipliers = ResourceRates{Gold: 1, Oil: 1, Steel: 1}
		}
		if c.Units == nil {
			c.Units = map[string]int{}
		}
		countryCopy := c
		countries[c.Name] = &countryCopy
		order = append(order, c.Name)
	}

	return countries, order, nil
}
