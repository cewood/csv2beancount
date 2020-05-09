package cmd

import (
	"os"

	"github.com/cewood/csv2beancount/internal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tplFile string

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert [CSV file to convert]",
	Short: "Convert a CSV file into Beancount (ledger like) format",
	Long: `This command takes a CSV file, and a config file describing some important
fields in that file, and then renders them in beancount (ledger like) format
using a builtin default template, or one provided via the command line.

This command does not alter any data in the file you provide, it simply reads
the file, then uses a template to transform that data and render it to stdout.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file, err := os.Open(args[0])

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  args[0],
			}).Fatal("error opening file")
		}

		internal.ProcessCsvFile(file, internal.GetConfig(), internal.GetTemplate(tplFile))
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(convertCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// convertCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// convertCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	convertCmd.PersistentFlags().StringVar(&tplFile, "template", "", "custom template file (to override the internal default one)")
}

// Typically this is in the root command, but since we don't actually
//  run the root command, that is have a Run property, then this wouldn't
//  as expected. So it's been moved here instead.
//
// initConfig reads in config file and ENV variables if set.
func initConfig() {
	internal.SetViperDefaults(cfgFile)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.WithFields(log.Fields{
			"file": viper.ConfigFileUsed(),
		}).Debug("config file loaded")
	} else {
		log.WithFields(log.Fields{
			"error": err,
		}).Debug("fatal error loading config file")
	}
}
