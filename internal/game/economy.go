package game

import "math"

// EconomyConfig holds every tunable number the turn engine uses to grow
// a country's population, treasury, and army. Keeping these in one
// struct (rather than scattered constants) is what makes balancing the
// game later a config change instead of a code change.
//
// Extending to a new resource (oil, steel, uranium, ...) later means:
//  1. add a rate field here (e.g. OilPerTerritoryPoint),
//  2. add the corresponding balance (e.g. Oil int) to Country,
//  3. add one line to Apply() computing and adding it.
// Nothing else in the engine or UI needs to change shape to support it.
type EconomyConfig struct {
	// PopulationGrowthRate is the fractional population growth applied
	// each turn, e.g. 0.01 for 1% growth per turn.
	PopulationGrowthRate float64

	// PopulationUnit and TerritoryUnit convert a country's raw
	// population/territory into "economic points" - this keeps the
	// income formulas readable instead of dealing in fractions of a
	// person. E.g. PopulationUnit=100000 means every 100,000 people is
	// one population point.
	PopulationUnit int64
	TerritoryUnit  int

	// Income/recruitment rates, expressed per economic point.
	GoldPerPopulationPoint   float64
	GoldPerTerritoryPoint    float64
	TroopsPerPopulationPoint float64
}

// DefaultEconomyConfig returns the starting balance used by a new game.
// These numbers were picked to feel reasonable against the seed data in
// data/countries.json (starting gold in the low thousands, troops in
// the low thousands to tens of thousands) - tune freely.
func DefaultEconomyConfig() EconomyConfig {
	return EconomyConfig{
		PopulationGrowthRate:     0.01,
		PopulationUnit:           1_000_000,
		TerritoryUnit:            10_000,
		GoldPerPopulationPoint:   1.0,
		GoldPerTerritoryPoint:    1.0,
		TroopsPerPopulationPoint: 0.5,
	}
}

// TurnResult is the set of deltas produced by applying one turn of
// economy to a single country.
type TurnResult struct {
	PopulationGrowth int64
	GoldEarned       int
	TroopsRecruited  int
}

// economyPoints converts a country's current population and territory
// into the "economic point" units EconomyConfig's rates are expressed
// in.
func economyPoints(c *Country, cfg EconomyConfig) (populationPoints, territoryPoints float64) {
	if cfg.PopulationUnit <= 0 {
		cfg.PopulationUnit = 1
	}
	if cfg.TerritoryUnit <= 0 {
		cfg.TerritoryUnit = 1
	}
	populationPoints = float64(c.Population) / float64(cfg.PopulationUnit)
	territoryPoints = float64(c.TerritorySize) / float64(cfg.TerritoryUnit)
	return
}

// ProjectIncome computes what a country's gold income and troop
// recruitment WOULD be this turn, without mutating anything. This is
// what the "stats" panel uses to show "Gold Income / Turn" even before
// the player has ended a turn.
func ProjectIncome(c *Country, cfg EconomyConfig) (goldIncome int, troopIncome int) {
	popPoints, territoryPoints := economyPoints(c, cfg)
	goldIncome = int(math.Round(popPoints*cfg.GoldPerPopulationPoint + territoryPoints*cfg.GoldPerTerritoryPoint))
	troopIncome = int(math.Round(popPoints * cfg.TroopsPerPopulationPoint))
	return
}

// Apply runs one turn of economy for a single country: grows
// population, then computes and adds gold/troop income based on the
// (already grown) population and existing territory. It mutates c and
// returns the deltas that were applied, so callers (like the turn
// summary UI) can report exactly what happened without recomputing it.
//
// Every country - player or AI - goes through this same function, so
// there is exactly one place economic rules live.
func Apply(c *Country, cfg EconomyConfig) TurnResult {
	oldPop := c.Population
	growthFactor := 1 + cfg.PopulationGrowthRate
	newPop := int64(math.Round(float64(oldPop) * growthFactor))
	if newPop < oldPop {
		newPop = oldPop // guard against a misconfigured negative rate shrinking population below itself unexpectedly
	}
	c.Population = newPop

	goldEarned, troopsRecruited := ProjectIncome(c, cfg)
	c.Gold += goldEarned
	c.Troops += troopsRecruited

	return TurnResult{
		PopulationGrowth: newPop - oldPop,
		GoldEarned:       goldEarned,
		TroopsRecruited:  troopsRecruited,
	}
}
