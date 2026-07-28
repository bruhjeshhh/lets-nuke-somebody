// Territoria is a turn-based grand strategy game played in the terminal.
//
// Phase 1 (World Setup) covers loading the world from data, letting the
// player pick a starting country, and a bare-bones turn counter. No
// combat, economy, diplomacy, or AI behavior is implemented yet - see
// internal/game for the pieces that will grow in later phases.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"territoria/internal/game"
	"territoria/internal/tui"
)

func main() {
	dataPath := flag.String("data", "data/countries.json", "path to the country data JSON file")
	flag.Parse()

	countries, order, err := game.LoadCountries(*dataPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load world: %v\n", err)
		os.Exit(1)
	}

	state := game.NewState(countries, order)
	model := tui.New(state)

	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running game: %v\n", err)
		os.Exit(1)
	}
}
