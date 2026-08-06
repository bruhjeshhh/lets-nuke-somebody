package game

import (
	"fmt"
	"math"
	"strings"
)

// CombatConfig holds every tunable number the recruitment/combat system
// uses, same philosophy as EconomyConfig: balancing is a config change,
// not a code change.
type CombatConfig struct {
	// GoldPerTroop is the cost to manually recruit one troop via the
	// 'recruit' command.
	GoldPerTroop float64

	// WinnerCasualtyRate and LoserCasualtyRate are the fraction of
	// committed troops each side loses in a battle, regardless of
	// outcome - the winner still takes losses, just fewer.
	WinnerCasualtyRate float64
	LoserCasualtyRate  float64

	// AIRecruitGoldFraction is the fraction of its gold stockpile an
	// AI country automatically spends recruiting troops each turn.
	AIRecruitGoldFraction float64
}

// DefaultCombatConfig returns the starting balance used by a new game.
func DefaultCombatConfig() CombatConfig {
	return CombatConfig{
		GoldPerTroop:          10.0,
		WinnerCasualtyRate:    0.10,
		LoserCasualtyRate:     0.40,
		AIRecruitGoldFraction: 0.30,
	}
}

// CombatStrength is a single country's military strength for battle
// resolution purposes. Phase 3 keeps this deliberately simple - troop
// count only, per spec - but it's factored into its own function
// specifically so a later phase can fold in unit-based attack/defense
// (State.MilitaryPower, added in Phase 4) without touching Attack's
// control flow at all.
func CombatStrength(c *Country) int {
	return c.Troops
}

// RecruitTroops manually recruits `amount` troops for the named country
// at CombatConfig.GoldPerTroop gold each, deducting the full cost
// immediately. Used by the player-facing 'recruit' command; AI
// countries recruit automatically instead (see State.EndTurn).
func (s *State) RecruitTroops(countryName string, amount int) (cost int, err error) {
	if amount <= 0 {
		return 0, fmt.Errorf("amount must be a positive number")
	}

	c, ok := s.FindCountry(countryName)
	if !ok {
		return 0, fmt.Errorf("no country named %q", countryName)
	}

	cost = int(math.Round(float64(amount) * s.Combat.GoldPerTroop))
	if c.Gold < cost {
		return 0, fmt.Errorf("not enough gold: need %d, have %d", cost, c.Gold)
	}

	c.Gold -= cost
	c.Troops += amount
	return cost, nil
}

// aiAutoRecruit spends a fraction of every AI-controlled country's gold
// on troops each turn. Called once per turn from EndTurn, after the
// economy has already run - so AI recruitment is funded by that turn's
// income, same as a player deciding what to do with their gold.
func (s *State) aiAutoRecruit() {
	for _, name := range s.Order {
		c := s.Countries[name]
		if c.Owner != OwnerAI {
			continue
		}
		spend := int(float64(c.Gold) * s.Combat.AIRecruitGoldFraction)
		if spend <= 0 || s.Combat.GoldPerTroop <= 0 {
			continue
		}
		troops := int(float64(spend) / s.Combat.GoldPerTroop)
		if troops <= 0 {
			continue
		}
		cost := int(math.Round(float64(troops) * s.Combat.GoldPerTroop))
		c.Gold -= cost
		c.Troops += troops
	}
}

// BattleReport is the full outcome of one Attack call - everything the
// UI needs to render a battle summary.
type BattleReport struct {
	Attacker string
	Defender string

	AttackerTroopsCommitted int
	DefenderTroopsCommitted int

	AttackerCasualties int
	DefenderCasualties int

	AttackerTroopsRemaining int
	DefenderTroopsRemaining int

	Winner string // Attacker or Defender's name

	// TerritoryCaptured is the defender's territory size if the
	// attacker won, or 0 if the attack was repelled.
	TerritoryCaptured int
}

// PlayerOwnedCountries returns every country currently owned by the
// player - the original starting country plus anything conquered since
// - in stable display order.
func (s *State) PlayerOwnedCountries() []*Country {
	var owned []*Country
	for _, name := range s.Order {
		c := s.Countries[name]
		if c.Owner == OwnerPlayer {
			owned = append(owned, c)
		}
	}
	return owned
}

// isNeighbor reports whether b's name appears in a's neighbor list,
// case-insensitively.
func isNeighbor(a *Country, bName string) bool {
	target := strings.ToLower(bName)
	for _, n := range a.Neighbors {
		if strings.ToLower(n) == target {
			return true
		}
	}
	return false
}

// addNeighbor appends name to a's neighbor list if it isn't already
// present (case-insensitively) and isn't the country itself.
func addNeighbor(a *Country, name string) {
	if strings.EqualFold(a.Name, name) || isNeighbor(a, name) {
		return
	}
	a.Neighbors = append(a.Neighbors, name)
}

// Attack resolves one battle: the player's country (Attacker is always
// State.PlayerCountry - AI countries don't initiate attacks in this
// phase) against the named defender. Only a country adjacent to the
// player's home country can be attacked. Combat strength is
// CombatStrength (troop count); both sides take casualties regardless
// of outcome, and a winning attacker gains the defender's territory and
// absorbs its neighbor connections, so conquest can chain outward
// through captured ground on future attacks.
func (s *State) Attack(defenderName string) (BattleReport, error) {
	if !s.PlayerHasSelected() {
		return BattleReport{}, fmt.Errorf("you haven't selected a country yet")
	}

	attacker := s.Countries[s.PlayerCountry]

	defender, ok := s.FindCountry(defenderName)
	if !ok {
		return BattleReport{}, fmt.Errorf("no country named %q", defenderName)
	}
	if strings.EqualFold(defender.Name, attacker.Name) {
		return BattleReport{}, fmt.Errorf("you can't attack your own country")
	}
	if defender.Owner == OwnerPlayer {
		return BattleReport{}, fmt.Errorf("%s is already under your control", defender.Name)
	}
	if !isNeighbor(attacker, defender.Name) {
		return BattleReport{}, fmt.Errorf("%s does not border %s and cannot be attacked directly", defender.Name, attacker.Name)
	}

	attackerTroops := CombatStrength(attacker)
	defenderTroops := CombatStrength(defender)

	report := BattleReport{
		Attacker:                attacker.Name,
		Defender:                defender.Name,
		AttackerTroopsCommitted: attackerTroops,
		DefenderTroopsCommitted: defenderTroops,
	}

	attackerWins := attackerTroops > defenderTroops // defender wins ties

	if attackerWins {
		report.AttackerCasualties = int(math.Round(float64(attackerTroops) * s.Combat.WinnerCasualtyRate))
		report.DefenderCasualties = int(math.Round(float64(defenderTroops) * s.Combat.LoserCasualtyRate))
		report.Winner = attacker.Name
	} else {
		report.AttackerCasualties = int(math.Round(float64(attackerTroops) * s.Combat.LoserCasualtyRate))
		report.DefenderCasualties = int(math.Round(float64(defenderTroops) * s.Combat.WinnerCasualtyRate))
		report.Winner = defender.Name
	}

	attacker.Troops = clampNonNegative(attacker.Troops - report.AttackerCasualties)
	defender.Troops = clampNonNegative(defender.Troops - report.DefenderCasualties)
	report.AttackerTroopsRemaining = attacker.Troops
	report.DefenderTroopsRemaining = defender.Troops

	if attackerWins {
		report.TerritoryCaptured = defender.TerritorySize
		defender.Owner = OwnerPlayer

		// The attacker's reach grows through captured ground: anything
		// that bordered the defender is now reachable from the
		// player's empire on a future attack.
		for _, n := range defender.Neighbors {
			addNeighbor(attacker, n)
		}
	}

	return report, nil
}

func clampNonNegative(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
