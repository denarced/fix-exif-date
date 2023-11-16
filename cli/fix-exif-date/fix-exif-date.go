// Package main implements the main CLI.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/denarced/fix-exif-date/lib/fixexif"
	"github.com/denarced/fix-exif-date/shared"
)

// CLI for this application.
var CLI struct {
	Timezone string   `help:"Override local timezone." default:"Local"`
	Files    []string `arg:"" name:"file" help:"Photos to fix."`
}

func main() {
	shared.InitLogging()
	shared.Logger.Info().Msg(" ------ Start ----- ")
	kong.Parse(&CLI)
	location := parseTimezone(CLI.Timezone)
	if location == nil {
		os.Exit(2)
	}
	out := &cliOutput{first: true, indent: 4}
	for _, each := range CLI.Files {
		err := fixexif.FixDate(each, location, out)
		if err != nil {
			shared.Logger.Error().
				Str("filepath", each).
				Err(err).
				Msg("Failed to fix EXIF date. Quitting.")
			os.Exit(1)
		}
	}
	shared.Logger.Info().Msg(" ----- Done ----- ")
}

func parseTimezone(timezone string) *time.Location {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		shared.Logger.Info().Err(err).Msg("Failed to load timezone.")
		fmt.Fprintf(os.Stderr, "Invalid timezone location: %s\n", timezone)
		return nil
	}
	return location
}

type cliOutput struct {
	first  bool
	indent int
}

func (v *cliOutput) PrintFile(file string) {
	if !v.first {
		fmt.Println()
	}
	v.first = false
	fmt.Printf(" -- %s\n", file)
}

func (v *cliOutput) SkipFile() {
	fmt.Printf("%sskip\n", strings.Repeat(" ", v.indent))
}

func (v *cliOutput) PrintDates(tag uint16, original, updated string) {
	v.printPair(tag, original, updated)
}

func (v *cliOutput) PrintOffsets(tag uint16, original, updated string) {
	v.printPair(tag, original, updated)
}

func (v *cliOutput) printPair(tag uint16, first, second string) {
	fmt.Printf("%s0x%04x %s -> %s\n", v.createIndent(), tag, first, second)
}

func (v *cliOutput) createIndent() string {
	return strings.Repeat(" ", v.indent)
}

func (v *cliOutput) Done(success bool) {
	outcome := "ok"
	if !success {
		outcome = "fail"
	}
	fmt.Printf("%sdone: %s\n", v.createIndent(), outcome)
}
