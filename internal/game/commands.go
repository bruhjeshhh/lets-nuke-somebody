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
	"  end               End the current turn and advance to the next",
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

	case "end":
		turn := s.EndTurn()
		return CommandResult{Lines: []string{fmt.Sprintf("Turn ended. It is now turn %d.", turn)}}

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

	neighbors := "none"
	if len(c.Neighbors) > 0 {
		neighbors = strings.Join(c.Neighbors, ", ")
	}

	return []string{
		fmt.Sprintf("=== %s ===", c.Name),
		fmt.Sprintf("Population:     %d", c.Population),
		fmt.Sprintf("Territory Size: %d km^2", c.TerritorySize),
		fmt.Sprintf("Gold:           %d", c.Gold),
		fmt.Sprintf("Troops:         %d", c.Troops),
		fmt.Sprintf("Owner:          %s", c.Owner),
		fmt.Sprintf("Neighbors:      %s", neighbors),
		fmt.Sprintf("Current Turn:   %d", s.Turn),
	}
}
