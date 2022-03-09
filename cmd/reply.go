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

	c, subject, err := createResponseCalendar(e, r, from)
	if err != nil {
		log.Fatal(err)
	}
	body, err := createResponseEmail(c, from, subject)
	if err != nil {
		log.Fatal(err)
	}
	to := extractMailAddresses(m.From)
	if toEmail != "" {
		to = []string{toEmail}
	}

	if dry {
		printMail(from, to, body)
	} else {
		err := sendMail(from, to, body)
		if err != nil {
			log.Fatal(err)
		}
		_ = deleteICSFile(e.GetProperty(ics.ComponentPropertyUniqueId).Value)
	}
}

func createResponseCalendar(e *ics.VEvent, reply ics.PropertyParameter, email string) (*ics.Calendar, string, error) {
	c := ics.NewCalendar()
	c.SetMethod(ics.MethodReply)
	// c.SetProductId("github.com/matoubidou/mutt-itip")
	c.SetProductId("Microsoft Exchange Server 2010")

	copyProperty := func(dst, src *ics.VEvent, p ics.ComponentProperty) {
		prop := src.GetProperty(p)
		if prop == nil {
			log.Debugf("event w/o property %v. skip copying the property..", p)
			return
		}
		dst.AddProperty(p, prop.Value)
	}

	p := e.GetProperty(ics.ComponentPropertyUniqueId)
	if p == nil {
		return nil, "", fmt.Errorf("event w/o uid")
	}
	uid := p.Value
	resEvent := c.AddEvent(uid)
	resEvent.AddAttendee(email, reply)
	resEvent.AddProperty(ics.ComponentPropertyDtstamp, time.Now().UTC().Format(dateLayout)+"Z")
	copyProperty(resEvent, e, ics.ComponentPropertyOrganizer)
	summary := uid
	p = e.GetProperty(ics.ComponentPropertySummary)
	if p != nil {
		summary = p.Value
	}
	summary = fmt.Sprintf("%s: %s", reply, summary)
	resEvent.SetProperty(ics.ComponentPropertySummary, summary)
	if replyComment != "" {
		resEvent.AddProperty(ics.ComponentProperty(ics.PropertyComment), replyComment)
	}

	// TODO: this seems odd
	// copyProperty(resEvent, e, ics.ComponentProperty(ics.PropertyRecurrenceId))
	copyProperty(resEvent, e, ics.ComponentPropertySequence)

	// according to rfc not needed but trying to get things working
	// with Microsoft Exchange Server 2010
	copyProperty(resEvent, e, ics.ComponentPropertyDtStart)
	copyProperty(resEvent, e, ics.ComponentPropertyDtEnd)
	copyProperty(resEvent, e, ics.ComponentPropertyTransp)
	copyProperty(resEvent, e, ics.ComponentPropertyStatus)
	copyProperty(resEvent, e, ics.ComponentPropertyClass)
	copyProperty(resEvent, e, ics.ComponentPropertyCategories)
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-CDO-APPT-SEQUENCE"), "0")
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-CDO-OWNERAPPTID"), "2120437062")
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-CDO-BUSYSTATUS"), "BUSY")
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-CDO-INTENDEDSTATUS"), "BUSY")
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-CDO-ALLDAYEVENT"), "FALSE")
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-CDO-IMPORTANCE"), "1")
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-CDO-INSTTYPE"), "0")
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-DONOTFORWARDMEETING"), "FALSE")
	// resEvent.AddProperty(ics.ComponentProperty("X-MICROSOFT-DISALLOW-COUNTER"), "FALSE")

	return c, fmt.Sprintf("Subject: %s", summary), nil
}

func createResponseEmail(ical *ics.Calendar, from string, subject string) ([]byte, error) {
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("From: %s\n", from))
	body.WriteString(fmt.Sprintf("Subject: %s\n", subject))
	body.WriteString("MIME-Version: 1.0\n")
	body.WriteString("Content-Type: text/calendar; charset=utf-8; method=reply\n")
	body.WriteString("Content-Transfer-Encoding: quoted-printable\n\n")

	err := ical.SerializeTo(&body)
	if err != nil {
		return nil, err
	}

	return body.Bytes(), nil
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
		return fmt.Errorf("execution of smtpPassCmd '%s' returned: %w, stderr: %v", smtpPassCmd, err, stderr.String())
	}
	smtpPass := strings.Trim(string(b), "\n\t ")
	host := strings.Split(smtpAddr, ":")[0]

	err = smtp.SendMail(smtpAddr, smtp.PlainAuth("", smtpUser, smtpPass, host), from, to, body)
	if err != nil {
		return err
	}
	return nil
}
