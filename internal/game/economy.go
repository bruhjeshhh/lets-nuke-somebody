package game

import "math"

// ResourceRates expresses a relative strength per resource. It's used
// two ways in this file: as a per-country multiplier on the base income
// formulas (ResourceMultipliers on Country - a landlocked industrial
// power might be {Gold: 1.0, Oil: 0.3, Steel: 1.5}), and conceptually as
// the shape any future per-resource rate takes.
type ResourceRates struct {
	Gold  float64 `json:"gold"`
	Oil   float64 `json:"oil"`
	Steel float64 `json:"steel"`
}

// EconomyConfig holds every tunable number the turn engine uses to grow
// a country's population, treasury, resources, and army. Keeping these
// in one struct (rather than scattered constants) is what makes
// balancing the game later a config change instead of a code change.
//
// Extending to a brand new resource (uranium, food, ...) later means:
//  1. add base rate fields here (e.g. UraniumPerTerritoryPoint),
//  2. add the corresponding multiplier field to ResourceRates and
//     balance field (e.g. Uranium int) to Country,
//  3. add one line to ProjectIncome/Apply computing and adding it.
//
// Nothing else in the engine or UI needs to change shape to support it.
type EconomyConfig struct {
	// PopulationGrowthRate is the fractional population growth applied
	// each turn, e.g. 0.01 for 1% growth per turn.
	PopulationGrowthRate float64

	// PopulationUnit and TerritoryUnit convert a country's raw
	// population/territory into "economic points" - this keeps the
	// income formulas readable instead of dealing in fractions of a
	// person. E.g. PopulationUnit=1_000_000 means every million people
	// is one population point.
	PopulationUnit int64
	TerritoryUnit  int

	// Base income/recruitment rates, expressed per economic point,
	// BEFORE a country's own ResourceMultipliers are applied. Troop
	// recruitment is deliberately not scaled by ResourceMultipliers -
	// it stays population-only, independent of a country's industrial
	// profile.
	GoldPerPopulationPoint   float64
	GoldPerTerritoryPoint    float64
	OilPerPopulationPoint    float64
	OilPerTerritoryPoint     float64
	SteelPerPopulationPoint  float64
	SteelPerTerritoryPoint   float64
	TroopsPerPopulationPoint float64
}

// DefaultEconomyConfig returns the starting balance used by a new game.
// These numbers were picked to feel reasonable against the seed data in
// data/countries.json (starting treasuries in the low thousands, troops
// in the low thousands to tens of thousands) - tune freely.
func DefaultEconomyConfig() EconomyConfig {
	return EconomyConfig{
		PopulationGrowthRate:     0.01,
		PopulationUnit:           1_000_000,
		TerritoryUnit:            10_000,
		GoldPerPopulationPoint:   1.0,
		GoldPerTerritoryPoint:    1.0,
		OilPerPopulationPoint:    0.4,
		OilPerTerritoryPoint:     0.4,
		SteelPerPopulationPoint:  0.4,
		SteelPerTerritoryPoint:   0.4,
		TroopsPerPopulationPoint: 0.5,
	}
}

// IncomePreview is one turn's worth of economic output for a country:
// what Apply will add to its stockpiles/army if run right now. Exposed
// separately from Apply so the UI can show "income per turn" without
// mutating anything or waiting for the player to end a turn.
type IncomePreview struct {
	Gold   int
	Oil    int
	Steel  int
	Troops int
}

// TurnResult is the set of deltas produced by applying one turn of
// economy to a single country. It mirrors IncomePreview but is named
// separately since it represents something that already happened
// (past tense in the UI: "Gold earned: +X") rather than a projection.
type TurnResult struct {
	PopulationGrowth int64
	GoldEarned       int
	OilEarned        int
	SteelEarned      int
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

// ProjectIncome computes what a country's resource income and troop
// recruitment WOULD be this turn, without mutating anything. This is
// what the "stats"/"resources" panels use to show "income / turn" even
// before the player has ended a turn. Gold/Oil/Steel are each scaled by
// the country's own ResourceMultipliers, so a real-world oil producer
// nets more oil per population/territory point than a landlocked
// industrial power would, without any per-country special-casing in
// this function.
func ProjectIncome(c *Country, cfg EconomyConfig) IncomePreview {
	popPoints, territoryPoints := economyPoints(c, cfg)
	m := c.ResourceMultipliers

	gold := popPoints*cfg.GoldPerPopulationPoint + territoryPoints*cfg.GoldPerTerritoryPoint
	oil := popPoints*cfg.OilPerPopulationPoint + territoryPoints*cfg.OilPerTerritoryPoint
	steel := popPoints*cfg.SteelPerPopulationPoint + territoryPoints*cfg.SteelPerTerritoryPoint

	return IncomePreview{
		Gold:   int(math.Round(gold * m.Gold)),
		Oil:    int(math.Round(oil * m.Oil)),
		Steel:  int(math.Round(steel * m.Steel)),
		Troops: int(math.Round(popPoints * cfg.TroopsPerPopulationPoint)),
	}
}

// Apply runs one turn of economy for a single country: grows
// population, then computes and adds resource income/troop recruitment
// based on the (already grown) population and existing territory. It
// mutates c and returns the deltas that were applied, so callers (like
// the turn summary UI) can report exactly what happened without
// recomputing it.
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

	income := ProjectIncome(c, cfg)
	c.Gold += income.Gold
	c.Oil += income.Oil
	c.Steel += income.Steel
	c.Troops += income.Troops

	return TurnResult{
		PopulationGrowth: newPop - oldPop,
		GoldEarned:       income.Gold,
		OilEarned:        income.Oil,
		SteelEarned:      income.Steel,
		TroopsRecruited:  income.Troops,
	}
}
