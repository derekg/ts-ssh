// This file contains the original main.go implementation before fang CLI integration
// It's kept for reference and potential fallback scenarios

package main

// NOTE: This file is not compiled (no main function)
// Original implementation moved to main_helpers.go and cli.go

/*
Original main.go content:
- Used Go standard flag library
- Large main() function with many individual flags
- Direct argument parsing and command dispatching
- All logic inline in main() function

New implementation:
- Uses Charmbracelet fang CLI framework
- Structured subcommands (connect, scp, list, exec, multi, config, pqc, version)
- Modular command structure with individual Run() methods
- Better help output and usage documentation
- Backwards compatibility through automatic command detection
*/
