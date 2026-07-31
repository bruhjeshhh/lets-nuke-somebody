package game

import (
	"fmt"
	"sort"
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
	"  help              Show this list of commands",
	"  countries         List every country in the world and its owner",
	"  select <country>  Choose your starting country (one-time)",
	"  stats             Show detailed info for your country",
	"  leaderboard       Show the richest and strongest countries",
	"  end               End the current turn: economy runs, then advance",
	"  quit              Exit the game",
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
	goldIncome, troopIncome := ProjectIncome(c, s.Economy)

	neighbors := "none"
	if len(c.Neighbors) > 0 {
		neighbors = strings.Join(c.Neighbors, ", ")
	}

	return []string{
		fmt.Sprintf("=== %s ===", c.Name),
		fmt.Sprintf("Population:              %d", c.Population),
		fmt.Sprintf("Territory:               %d km^2", c.TerritorySize),
		fmt.Sprintf("Gold:                    %d", c.Gold),
		fmt.Sprintf("Gold Income / Turn:      +%d", goldIncome),
		fmt.Sprintf("Troops:                  %d", c.Troops),
		fmt.Sprintf("Troop Recruitment / Turn: +%d", troopIncome),
		fmt.Sprintf("Owner:                   %s", c.Owner),
		fmt.Sprintf("Neighbors:               %s", neighbors),
		fmt.Sprintf("Current Turn:            %d", s.Turn),
	}
}

// showLeaderboard renders the top countries by treasury and by troop
// count. Ties break alphabetically so results are deterministic.
func showLeaderboard(s *State) []string {
	const topN = 5

	lines := []string{fmt.Sprintf("=== Leaderboard (Turn %d) ===", s.Turn), "", "Richest countries (gold):"}
	for i, c := range s.RankedByGold() {
		if i >= topN {
			break
		}
		lines = append(lines, fmt.Sprintf("  %d. %-20s %d gold%s", i+1, c.Name, c.Gold, ownerTag(c)))
	}

	lines = append(lines, "", "Strongest countries (troops):")
	for i, c := range s.RankedByTroops() {
		if i >= topN {
			break
		}
		lines = append(lines, fmt.Sprintf("  %d. %-20s %d troops%s", i+1, c.Name, c.Troops, ownerTag(c)))
	}

	return lines
}

// showTurnSummary formats the result of State.EndTurn into readable
// event lines: turn advanced, then (if the player has a country)
// population/gold/troop deltas for that turn.
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
