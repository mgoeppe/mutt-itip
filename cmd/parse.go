package cmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/DusanKasan/parsemail"
	ics "github.com/arran4/golang-ical"
)

func extractMethod(c *ics.Calendar) ics.Method {
	for _, p := range c.CalendarProperties {
		if p.BaseProperty.IANAToken == string(ics.PropertyMethod) {
			return ics.Method(p.Value)
		}
	}
	return ""
}

func embeddedCalendarReader(r io.Reader) (io.Reader, *parsemail.Email, error) {
	m, err := parsemail.Parse(r)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse email: %w", err)
	}
	for _, a := range m.EmbeddedFiles {
		if strings.HasPrefix(a.ContentType, "text/calendar") {
			return a.Data, &m, nil
		}
	}
	return nil, nil, fmt.Errorf("no embedded text/calendar file found in mail")
}

func parseInput(r io.Reader) (*parsemail.Email, *ics.Calendar, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	r = bytes.NewReader(b)
	er, m, err := embeddedCalendarReader(r)
	if err == nil {
		r = er
	} else {
		r = bytes.NewReader(b)
	}

	c, err := ics.ParseCalendar(r)
	if err != nil {
		return nil, nil, err
	}
	return m, c, nil
}
