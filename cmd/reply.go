// this file implements the reply commands according to https://datatracker.ietf.org/doc/html/rfc5546#section-3.2.3
package cmd

import (
	"bytes"
	"fmt"
	"net/mail"
	"net/smtp"
	"os"
	"os/exec"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	replyComment string
	toEmail      string

	acceptCmd = &cobra.Command{
		Use:   "accept",
		Short: "accepts the invitation",
		Run: func(cmd *cobra.Command, args []string) {
			reply(ics.ParticipationStatusAccepted)
		},
	}

	declineCmd = &cobra.Command{
		Use:   "decline",
		Short: "declines the invitation",
		Run: func(cmd *cobra.Command, args []string) {
			reply(ics.ParticipationStatusDeclined)
		},
	}

	tentativeCmd = &cobra.Command{
		Use:   "tentative",
		Short: "tentatively accepts the invitation",
		Run: func(cmd *cobra.Command, args []string) {
			reply(ics.ParticipationStatusTentative)
		},
	}
)

func init() {
	addFlags := func(cmd *cobra.Command) *cobra.Command {
		cmd.Flags().StringVar(&replyComment, "replyComment", "", "the comment being sent along with the reply")
		cmd.Flags().StringVar(&toEmail, "to", "", "if set the reply will be sent to that mail address instead of replying to the sender")
		cmd.Flags().BoolVar(&dry, "dry", false, "if set reply is printed instead of being sent")
		return cmd
	}

	rootCmd.AddCommand(addFlags(acceptCmd))
	rootCmd.AddCommand(addFlags(declineCmd))
	rootCmd.AddCommand(addFlags(tentativeCmd))
}

func reply(r ics.PropertyParameter) {
	from := viper.GetString("mail")
	if from == "" {
		log.Fatalf("unable to sent reply: you need to set mail in %s", viper.ConfigFileUsed())
	}

	m, c, err := parseInput(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	// validate incoming calendar and event
	if extractMethod(c) != ics.MethodRequest {
		log.Fatal("replies are only allowed on ics with method REQUEST")
	}
	e := oneEventOrDie(c)

	c, subject := createResponseCalendar(e, r, from)
	body := createResponseEmail(c, subject)
	to := extractMailAddresses(m.From)
	if toEmail != "" {
		to = []string{toEmail}
	}

	log.Infof("replying to invitation with %s", r)
	printInvite("REQUEST", e, false)

	if dry {
		printMail(from, to, body)
		// that way we see the outputin mutt
		os.Exit(1)
	} else {
		err := sendMail(from, to, body)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func createResponseCalendar(e *ics.VEvent, reply ics.PropertyParameter, email string) (*ics.Calendar, string) {
	c := ics.NewCalendar()
	c.SetMethod(ics.MethodReply)

	copyProperty := func(dst, src *ics.VEvent, p ics.ComponentProperty) {
		prop := src.GetProperty(p)
		if prop == nil {
			log.Debugf("event w/o property %v. skip copying the property..", p)
			return
		}
		dst.AddProperty(p, prop.Value)
	}

	uid := e.GetProperty(ics.ComponentPropertyUniqueId).Value
	resEvent := c.AddEvent(uid)
	resEvent.AddAttendee(email, reply)
	resEvent.AddProperty(ics.ComponentPropertyDtstamp, time.Now().Format(dateLayout))
	copyProperty(resEvent, e, ics.ComponentPropertyOrganizer)
	// TODO: this seems odd
	copyProperty(resEvent, e, ics.ComponentProperty(ics.PropertyRecurrenceId))
	copyProperty(resEvent, e, ics.ComponentPropertySequence)
	// according to rfc not needed
	// copyProperty(resEvent, e, ics.ComponentPropertyDtStart)
	// copyProperty(resEvent, e, ics.ComponentPropertyDtEnd)

	if replyComment != "" {
		resEvent.AddProperty(ics.ComponentProperty(ics.PropertyComment), replyComment)
	}

	answer := uid
	summary := e.GetProperty(ics.ComponentPropertySummary)
	if summary != nil {
		answer = summary.Value
	}
	return c, fmt.Sprintf("Subject: %s: %s\n", fmt.Sprintf("%s", reply), answer)
}

func createResponseEmail(ical *ics.Calendar, subject string) []byte {
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("Subject: %s", subject))
	body.WriteString("MIME-Version: 1.0\n\n")

	body.WriteString("Content-Type: text/calendar; charset=utf-8; method=reply\n")
	body.WriteString("Content-Transfer-Encoding: quoted-printable\n\n")

	ical.SerializeTo(&body)

	return body.Bytes()
}

func extractMailAddresses(addrs []*mail.Address) []string {
	s := []string{}
	for _, a := range addrs {
		s = append(s, a.Address)
	}
	return s
}

func printMail(from string, to []string, body []byte) {
	log.Infof("would send the following mail (smtpAddr: %v, from: %v, to: %v):\n\n%v\n",
		viper.GetString("smtpAddr"),
		from,
		to,
		string(body),
	)
}

func sendMail(from string, to []string, body []byte) error {
	smtpAddr := viper.GetString("smtpAddr")
	if smtpAddr == "" {
		return fmt.Errorf("unable to sent reply: you need to set smtpAddr in %s", viper.ConfigFileUsed())
	}
	smtpUser := viper.GetString("smtpUser")
	smtpPassCmd := viper.GetString("smtpPassCmd")

	cmd := exec.Command("/bin/bash", "-c", smtpPassCmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("execution of smtpPassCmd '%s' returned: %w, stderr: %v", smtpPassCmd, err, string(stderr.Bytes()))
	}
	smtpPass := strings.Trim(string(b), "\n\t ")
	host := strings.Split(smtpAddr, ":")[0]

	err = smtp.SendMail(smtpAddr, smtp.PlainAuth("", smtpUser, smtpPass, host), from, to, body)
	if err != nil {
		return err
	}
	return nil
}
