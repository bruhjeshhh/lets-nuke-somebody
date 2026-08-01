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
