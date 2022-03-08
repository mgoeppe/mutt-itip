package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	prettytext "github.com/jedib0t/go-pretty/text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var formatCmd = &cobra.Command{
	Use:   "format",
	Short: "format displays an invitation",
	Long:  "format displays an invitation in an email given. if mail file is not given stdin is read",
	Run: func(cmd *cobra.Command, args []string) {
		_, c, err := parseInput(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		if len(c.Events()) == 0 {
			log.Fatal("no events found")
		}
		for _, e := range c.Events() {
			printInvite(string(extractMethod(c)), e, true)
		}
	},
}

func init() {
	rootCmd.AddCommand(formatCmd)
}

func printInvite(method string, e *ics.VEvent, includeDescription bool) {
	strip := func(s string) string {
		s = strings.TrimPrefix(s, "MAILTO:")
		s = strings.TrimPrefix(s, "mailto:")
		return s
	}
	formatTime := func(s string) string {
		t, err := time.ParseInLocation(dateLayout, s, time.Local)
		if err != nil {
			return s
		}
		return t.String()
	}

	printProperty(e, "Summary", ics.ComponentPropertySummary,
		func(s string) string {
			return s
		})
	printProperty(e, "Organizer", ics.ComponentPropertyOrganizer,
		func(s string) string {
			return strip(s)
		})
	printProperty(e, "Start", ics.ComponentPropertyDtStart,
		func(s string) string {
			return formatTime(s)
		})
	printProperty(e, "End", ics.ComponentPropertyDtEnd,
		func(s string) string {
			return formatTime(s)
		})
	printProperty(e, "Location", ics.ComponentPropertyLocation,
		func(s string) string {
			return strip(s)
		})
	fmt.Printf("%s %s\n", formatLabel("Method"), method)
	printProperty(e, "UID", ics.ComponentPropertyUniqueId,
		func(s string) string {
			return strip(s)
		})

	fmt.Println(formatLabel("Participants"))
	for _, a := range e.Attendees() {
		fmt.Printf("  * %s\n", strip(a.Email()))
	}

	if includeDescription {
		printProperty(e, "Description", ics.ComponentPropertyDescription,
			func(s string) string {
				s = strings.ReplaceAll(s, "\\n", "\n")
				s = strings.ReplaceAll(s, "\\,", ",")
				return "\n\n" + s
			})
	}
}

func printProperty(e *ics.VEvent, label string, p ics.ComponentProperty, m func(string) string) {
	prop := e.GetProperty(p)
	if prop != nil {
		fmt.Printf("%s %s\n", formatLabel(label), m(prop.Value))
	}
}

func formatLabel(label string) string {
	return fmt.Sprintf("%v:", prettytext.FgGreen.Sprint(prettytext.AlignLeft.Apply(label, 12)))
}
