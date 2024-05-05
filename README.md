This library offers quick form handling for user-facing workflows. Using the power of `huh.Form`, it makes it easy to display forms to the user and gather input. The forms must have embedded pointer values for each field to have access to the result. Calling the Interact function requires a function that generates the form, a pointer to the documentation string, and a closure that checks if the saved values are valid.

# Example

```go
package main


import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/pydpll/errorutils"
	"github.com/pydpll/flagmaker"
	"github.com/urfave/cli/v2"
)

// version vars
var (
	CommitID         string
	flagmakerVersion string
)

var (
	reference string
	progs     []string
)

func NewForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("From which species are these genomes derived?").
				Options(
					huh.NewOption("Cryptococcus neoformans", "H99"),
					huh.NewOption("Cryptococcus gattii", "WM127"),
					huh.NewOption("Candida albicans", "CS1345"),
				).
				Value(&reference),
			huh.NewMultiSelect[string]().Description("programs to choose").
				Options(
					huh.NewOptions[string]("sp", "tp", "ed", "ms")...,
				).Value(&progs),
		),
	)
}

//empty responses are invalid and trigger confirmation 
func invalidResponse(p *[]string, r *string) func() bool {
	return func() bool {
		progs = *p
		reference = *r
		return len(progs) == 0 || reference == ""
	}
}

func main() {

	app := &cli.App{
		Name:    "00001_flagmaker",
		Usage:   "interactive options selector of 00001",
		Version: fmt.Sprintf("commit:%s, flagmaker:%s", CommitID, flagmakerVersion),
		Action: func(ctx *cli.Context) error {
			doc := programNotes
			err := flagmaker.Interact(NewForm, &doc, invalidResponse(&progs, &reference))
			errorutils.ExitOnFail(err)
			return nil
		},
	}

	err := app.Run(os.Args)
	errorutils.ExitOnFail(err)
	//terminate early when version flag is set
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		os.Exit(0)
	}
    //this print statement lets you capture the flags. This means that the compiled program is used like a gum command using command substitution in the shell.
	fmt.Printf("-programs %s -GenomeREF %s",
		strings.Join(progs, ","),
		reference)
}

const programNotes string = `00001 - AVALANCHE
This workflow takes all the paired end genomes in a directory and assembles multiple times with different tools for subsequent reconciliation, scaffolding, and polishing.
The program assumes that all genomes belong to the same species and that there is only one reference genome for each species.`
```