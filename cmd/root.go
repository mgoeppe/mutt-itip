package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// we always parse in local
const dateLayout = "20060102T150405"

var (
	dry bool

	rootCmd = &cobra.Command{
		Use:   "mutt-itip",
		Short: "mutt-itip displays and replies to email invitations",
		Long:  "mutt-itip displays emails with embedded files of type text/calendar and is able to reply via email implementing Section 3.2.3 of RFC 5546.",
	}
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	viper.SetConfigName("mutt-itiprc")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/")
	viper.AddConfigPath("$HOME/.config/mutt")
	viper.AddConfigPath("/etc")
	err := viper.ReadInConfig()
	if err != nil {
		log.Warnf("only format subcommand will work: %v", err)
	}
}
