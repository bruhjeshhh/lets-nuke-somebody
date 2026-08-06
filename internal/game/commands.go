package game

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CommandResult is the outcome of executing one command: the lines of
// text to display, and whether the command should cause the program to
// exit (e.g. "quit").
type CommandResult struct {
	Lines []string
	Quit  bool
}

var helpText = []string{
	"Available commands:",
	"  help                    Show this list of commands",
	"  countries               List every country in the world and its owner",
	"  select <country>        Choose your starting country (one-time)",
	"  stats                   Show your country's full status, including military",
	"  resources               Show your resource stockpile and income per turn",
	"  build <unit> <amount>   Purchase units (infantry, tanks, artillery, fighters, destroyers)",
	"  recruit <amount>        Recruit troops with gold",
	"  attack <country>        Attack a neighboring country",
	"  map                     Show world ownership - your empire vs. everyone else",
	"  leaderboard             Show the richest and strongest countries",
	"  end                     End the current turn: economy runs, then advance",
	"  quit                    Exit the game",
}

// Execute parses a raw command line and applies it to the given state,
// returning the text to display to the user. Execute never panics on bad
// input - unrecognized commands and errors are simply reported back as
// output lines.
func Execute(s *State, line string) CommandResult {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return CommandResult{Lines: []string{""}}
	}

	cmd := strings.ToLower(fields[0])
	args := fields[1:]

	switch cmd {
	case "help", "?":
		return CommandResult{Lines: helpText}

	case "countries":
		return CommandResult{Lines: listCountries(s)}

	case "select":
		if len(args) == 0 {
			return CommandResult{Lines: []string{"Usage: select <country name>"}}
		}
		name := strings.Join(args, " ")
		c, err := s.SelectCountry(name)
		if err != nil {
			return CommandResult{Lines: []string{"Error: " + err.Error()}}
		}
		return CommandResult{Lines: []string{
			fmt.Sprintf("You are now in control of %s.", c.Name),
			"All other nations are now AI-controlled.",
			"Type 'stats' to see your country's details.",
		}}

	case "stats":
		return CommandResult{Lines: showStats(s)}

	case "resources":
		return CommandResult{Lines: showResources(s)}

	case "build":
		return CommandResult{Lines: buildUnits(s, args)}

	case "recruit":
		return CommandResult{Lines: recruitTroops(s, args)}

	case "attack":
		return CommandResult{Lines: attackCountry(s, args)}

	case "map":
		return CommandResult{Lines: showMap(s)}

	case "leaderboard":
		return CommandResult{Lines: showLeaderboard(s)}

	case "end":
		return CommandResult{Lines: showTurnSummary(s.EndTurn())}

	case "quit", "exit":
		return CommandResult{Lines: []string{"Goodbye."}, Quit: true}

	default:
		return CommandResult{Lines: []string{
			fmt.Sprintf("Unknown command: %q. Type 'help' for a list of commands.", cmd),
		}}
	}
}

func listCountries(s *State) []string {
	names := make([]string, len(s.Order))
	copy(names, s.Order)
	sort.Strings(names)

	lines := make([]string, 0, len(names)+1)
	lines = append(lines, fmt.Sprintf("Countries (%d):", len(names)))
	for _, n := range names {
		c := s.Countries[n]
		owner := "AI"
		if c.Owner == OwnerPlayer {
			owner = "YOU"
		}
		lines = append(lines, fmt.Sprintf("  %-20s [%s]", c.Name, owner))
	}
	return lines
}

func showStats(s *State) []string {
	if !s.PlayerHasSelected() {
		return []string{"You haven't selected a country yet. Use 'select <country>' first."}
	}
	c := s.Countries[s.PlayerCountry]
	income := ProjectIncome(c, s.Economy)
	attack, defense, maintenance := s.MilitaryPower(c)

	neighbors := "none"
	if len(c.Neighbors) > 0 {
		neighbors = strings.Join(c.Neighbors, ", ")
	}

	owned := s.PlayerOwnedCountries()
	ownedNames := make([]string, len(owned))
	for i, o := range owned {
		ownedNames[i] = o.Name
	}

	lines := []string{
		fmt.Sprintf("=== %s ===", c.Name),
		fmt.Sprintf("Population:          %d", c.Population),
		fmt.Sprintf("Territory:           %d km^2", c.TerritorySize),
		"",
		"--- Resources ---",
		fmt.Sprintf("Gold:  %-10d (+%d / turn)", c.Gold, income.Gold),
		fmt.Sprintf("Oil:   %-10d (+%d / turn)", c.Oil, income.Oil),
		fmt.Sprintf("Steel: %-10d (+%d / turn)", c.Steel, income.Steel),
		fmt.Sprintf("Troops: %d (+%d / turn)", c.Troops, income.Troops),
		"",
		"--- Military ---",
	}
	lines = append(lines, militaryInventoryLines(s, c)...)
	lines = append(lines,
		fmt.Sprintf("Total Attack Power:  %d", attack),
		fmt.Sprintf("Total Defense Power: %d", defense),
		fmt.Sprintf("Maintenance / Turn:  %s", formatBundle(maintenance)),
		"",
		fmt.Sprintf("Owner:               %s", c.Owner),
		fmt.Sprintf("Neighbors:           %s", neighbors),
		fmt.Sprintf("Countries you control (%d): %s", len(ownedNames), strings.Join(ownedNames, ", ")),
		fmt.Sprintf("Current Turn:        %d", s.Turn),
	)
	return lines
}

// showResources renders just the resource side of a country's stats -
// current stockpile and income per turn for gold, oil, and steel.
func showResources(s *State) []string {
	if !s.PlayerHasSelected() {
		return []string{"You haven't selected a country yet. Use 'select <country>' first."}
	}
	c := s.Countries[s.PlayerCountry]
	income := ProjectIncome(c, s.Economy)

	return []string{
		fmt.Sprintf("=== %s: Resources (Turn %d) ===", c.Name, s.Turn),
		fmt.Sprintf("Gold:  %-10d (+%d / turn)", c.Gold, income.Gold),
		fmt.Sprintf("Oil:   %-10d (+%d / turn)", c.Oil, income.Oil),
		fmt.Sprintf("Steel: %-10d (+%d / turn)", c.Steel, income.Steel),
	}
}

// militaryInventoryLines lists owned unit counts in catalog order,
// including zero-count entries so the player can see what's buildable.
func militaryInventoryLines(s *State, c *Country) []string {
	if len(s.UnitOrder) == 0 {
		return []string{"(no unit types loaded)"}
	}
	lines := make([]string, 0, len(s.UnitOrder))
	for _, id := range s.UnitOrder {
		unit := s.Units[id]
		lines = append(lines, fmt.Sprintf("  %-12s %d  (atk %d / def %d, cost %s)",
			unit.Name, c.Units[id], unit.Attack, unit.Defense, formatBundle(unit.Cost)))
	}
	return lines
}

// buildUnits handles "build <unit> <amount>": purchases the requested
// unit for the player's own country.
func buildUnits(s *State, args []string) []string {
	if !s.PlayerHasSelected() {
		return []string{"You haven't selected a country yet. Use 'select <country>' first."}
	}
	if len(args) != 2 {
		return []string{"Usage: build <unit> <amount> (e.g. 'build tanks 5')"}
	}

	unitName := args[0]
	amount, err := strconv.Atoi(args[1])
	if err != nil {
		return []string{fmt.Sprintf("Amount must be a whole number, got %q.", args[1])}
	}

	unit, cost, err := s.BuildUnits(s.PlayerCountry, unitName, amount)
	if err != nil {
		return []string{"Error: " + err.Error()}
	}

	return []string{
		fmt.Sprintf("Built %d %s for %s.", amount, unit.Name, formatBundle(cost)),
		fmt.Sprintf("%s now has %d %s.", s.PlayerCountry, s.Countries[s.PlayerCountry].Units[unit.ID], unit.Name),
	}
}

// recruitTroops handles "recruit <amount>": buys troops for the
// player's own country with gold.
func recruitTroops(s *State, args []string) []string {
	if !s.PlayerHasSelected() {
		return []string{"You haven't selected a country yet. Use 'select <country>' first."}
	}
	if len(args) != 1 {
		return []string{"Usage: recruit <amount> (e.g. 'recruit 500')"}
	}

	amount, err := strconv.Atoi(args[0])
	if err != nil {
		return []string{fmt.Sprintf("Amount must be a whole number, got %q.", args[0])}
	}

	cost, err := s.RecruitTroops(s.PlayerCountry, amount)
	if err != nil {
		return []string{"Error: " + err.Error()}
	}

	c := s.Countries[s.PlayerCountry]
	return []string{
		fmt.Sprintf("Recruited %d troops for %d gold.", amount, cost),
		fmt.Sprintf("%s now has %d troops and %d gold.", c.Name, c.Troops, c.Gold),
	}
}

// attackCountry handles "attack <country>": launches an attack from the
// player's country against a named, adjacent target.
func attackCountry(s *State, args []string) []string {
	if !s.PlayerHasSelected() {
		return []string{"You haven't selected a country yet. Use 'select <country>' first."}
	}
	if len(args) == 0 {
		return []string{"Usage: attack <country>"}
	}

	target := strings.Join(args, " ")
	report, err := s.Attack(target)
	if err != nil {
		return []string{"Error: " + err.Error()}
	}
	return formatBattleReport(report)
}

func formatBattleReport(r BattleReport) []string {
	lines := []string{
		"=== Battle Report ===",
		fmt.Sprintf("Attacker: %-20s troops committed: %d", r.Attacker, r.AttackerTroopsCommitted),
		fmt.Sprintf("Defender: %-20s troops committed: %d", r.Defender, r.DefenderTroopsCommitted),
		"",
		fmt.Sprintf("Casualties - %s: %d, %s: %d", r.Attacker, r.AttackerCasualties, r.Defender, r.DefenderCasualties),
		fmt.Sprintf("Troops remaining - %s: %d, %s: %d", r.Attacker, r.AttackerTroopsRemaining, r.Defender, r.DefenderTroopsRemaining),
		"",
		fmt.Sprintf("Winner: %s", r.Winner),
	}
	if r.Winner == r.Attacker {
		lines = append(lines, fmt.Sprintf("Territory captured: %d km^2. %s is now under your control.", r.TerritoryCaptured, r.Defender))
	} else {
		lines = append(lines, fmt.Sprintf("The attack was repelled - %s holds its territory.", r.Defender))
	}
	return lines
}

// showMap renders a text-based view of world ownership: the player's
// empire first, then the rest of the world, each with territory size -
// there's no geometric/pixel map in this CLI, so this is the "does the
// map reflect new ownership" view.
func showMap(s *State) []string {
	owned := s.PlayerOwnedCountries()
	lines := []string{fmt.Sprintf("=== World Map (Turn %d) ===", s.Turn), ""}

	if len(owned) == 0 {
		lines = append(lines, "You don't control any territory yet - use 'select <country>' to begin.")
	} else {
		totalTerritory := 0
		lines = append(lines, "Your empire:")
		for _, c := range owned {
			lines = append(lines, fmt.Sprintf("  %-20s %d km^2", c.Name, c.TerritorySize))
			totalTerritory += c.TerritorySize
		}
		lines = append(lines, fmt.Sprintf("  (total: %d km^2 across %d countries)", totalTerritory, len(owned)))
	}

	lines = append(lines, "", "Rest of the world:")
	names := make([]string, len(s.Order))
	copy(names, s.Order)
	sort.Strings(names)
	for _, n := range names {
		c := s.Countries[n]
		if c.Owner == OwnerPlayer {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %-20s %d km^2 [AI]", c.Name, c.TerritorySize))
	}

	return lines
}

// showLeaderboard renders the top countries by treasury, standing army
// (troops), and unit-based military power. Ties break alphabetically so
// results are deterministic.
func showLeaderboard(s *State) []string {
	const topN = 5

	lines := []string{fmt.Sprintf("=== Leaderboard (Turn %d) ===", s.Turn), "", "Richest countries (gold):"}
	for i, c := range s.RankedByGold() {
		if i >= topN {
			break
		}
		lines = append(lines, fmt.Sprintf("  %d. %-20s %d gold%s", i+1, c.Name, c.Gold, ownerTag(c)))
	}

	lines = append(lines, "", "Largest standing armies (troops):")
	for i, c := range s.RankedByTroops() {
		if i >= topN {
			break
		}
		lines = append(lines, fmt.Sprintf("  %d. %-20s %d troops%s", i+1, c.Name, c.Troops, ownerTag(c)))
	}

	lines = append(lines, "", "Strongest militaries (unit power):")
	for i, c := range s.RankedByMilitaryPower() {
		if i >= topN {
			break
		}
		attack, defense, _ := s.MilitaryPower(c)
		lines = append(lines, fmt.Sprintf("  %d. %-20s %d power (atk %d / def %d)%s",
			i+1, c.Name, attack+defense, attack, defense, ownerTag(c)))
	}

	return lines
}

// showTurnSummary formats the result of State.EndTurn into readable
// event lines: turn advanced, then (if the player has a country)
// population/resource/troop deltas for that turn.
func showTurnSummary(summary TurnSummary) []string {
	lines := []string{fmt.Sprintf("Turn advanced. It is now turn %d.", summary.Turn)}

	if summary.PlayerCountry == "" {
		lines = append(lines, "(Select a country with 'select <country>' to see your own turn events.)")
		return lines
	}

	r := summary.Result
	lines = append(lines,
		fmt.Sprintf("Population increased by %d.", r.PopulationGrowth),
		fmt.Sprintf("Gold earned: +%d.", r.GoldEarned),
		fmt.Sprintf("Oil earned: +%d.", r.OilEarned),
		fmt.Sprintf("Steel earned: +%d.", r.SteelEarned),
		fmt.Sprintf("Troops recruited: +%d.", r.TroopsRecruited),
	)
	return lines
}

func ownerTag(c *Country) string {
	if c.Owner == OwnerPlayer {
		return "  [YOU]"
	}
	return ""
}

// formatBundle renders a ResourceBundle compactly, omitting any
// resource that's zero so unit costs that don't touch e.g. oil don't
// clutter the line with "oil 0".
func formatBundle(b ResourceBundle) string {
	if b.IsZero() {
		return "free"
	}
	var parts []string
	if b.Gold != 0 {
		parts = append(parts, fmt.Sprintf("%d gold", b.Gold))
	}
	if b.Oil != 0 {
		parts = append(parts, fmt.Sprintf("%d oil", b.Oil))
	}
	if b.Steel != 0 {
		parts = append(parts, fmt.Sprintf("%d steel", b.Steel))
	}
	return strings.Join(parts, ", ")
}
