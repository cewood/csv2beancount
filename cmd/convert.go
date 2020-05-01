package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CsvConfig is the config for parsing the csv file
// TODO: move this to internal package
type CsvConfig struct {
	AmountIn          int    // The amount in field index
	AmountOut         int    // The amount out field index
	Currency          string // The currency to use
	Date              int    // The date field index
	DateLayoutIn      string // The parsing format
	DateLayoutOut     string // The date output format
	DefaultAccount    string // The default account for transactions if no rule matches
	Description       int    // The description field index
	Fields            int    // Validate no. of fields; -1 is no check, 0 is infer from first row, and > 0 is explicit length
	Payee             int    // The payee field index
	ProcessingAccount string // The account this export/CSV pertains to
	Separator         rune   // The csv file separator
	Skip              int    // The number of csv rows to skip, excluding blank lines
}

// Config represents the config
// TODO: move this to internal package
type Config struct {
	Csv               CsvConfig
	TransactionsRules map[string]map[string]string
}

// Record represents a financial transaction record
// TODO: move this to internal package
type Record struct {
	AccountIn   string // The account in
	AccountOut  string // The acocunt out
	AmountIn    string // The amount in
	AmountOut   string // The amount out
	Comment     string // The comment, if provided
	Currency    string // The currency
	Date        string // The date
	Description string // The description, if present
	Payee       string // The payee
	Raw         string // The raw csv record
}

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
		processCsvFile(args[0], getConfig())
	},
}

// TODO: move this to internal package
func getConfig() Config {
	return Config{
		Csv: CsvConfig{
			AmountIn:          viper.GetInt("csv.amount_in"),
			AmountOut:         viper.GetInt("csv.amount_out"),
			Currency:          viper.GetString("csv.currency"),
			Date:              viper.GetInt("csv.date"),
			DateLayoutIn:      viper.GetString("csv.date_layout_in"),
			DateLayoutOut:     viper.GetString("csv.date_layout_out"),
			DefaultAccount:    viper.GetString("csv.default_account"),
			Description:       viper.GetInt("csv.description"),
			Fields:            viper.GetInt("csv.fields"),
			Payee:             viper.GetInt("csv.payee"),
			ProcessingAccount: viper.GetString("csv.processing_account"),
			Separator:         []rune(viper.GetString("csv.separator"))[0],
			Skip:              viper.GetInt("csv.skip"),
		},
		TransactionsRules: getTransactionsRules(viper.GetStringMapString("transactions_rules")),
	}
}

// TODO: move this to internal package
func getTransactionsRules(keys map[string]string) (rules map[string]map[string]string) {
	rules = make(map[string]map[string]string)

	for key := range keys {
		rules[key] = viper.GetStringMapString(fmt.Sprintf("transactions_rules.%s", key))
	}

	return rules
}

// TODO: move this to internal package
func processCsvFile(targetFile string, config Config) {
	f, _ := os.Open(targetFile)

	r := csv.NewReader(f)

	r.Comma = config.Csv.Separator

	// Force this setting initially, after skipping any records it's
	//  updated with the user provided value or the default of 0
	r.FieldsPerRecord = -1

	// Lines to skip at beginng of file, not including blank lines
	skip := config.Csv.Skip

	for {
		record, err := r.Read()

		// Skip any lines requested
		if skip > 0 {
			skip = skip - 1

			if skip == 0 {
				r.FieldsPerRecord = config.Csv.Fields
			}

			continue
		}

		// Check for errors
		switch err {
		case nil:
		case io.EOF:
			break
		default:
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("error while reading csv file")
		}

		parseCsvRecord(record, config)
	}
}

// TODO: move this to internal package
func parseCsvRecord(record []string, config Config) {
	const recordTemplate = `{{.Date}} * "{{.Payee}}" "{{.Description}}"
  ; {{ .Raw }}
  {{.AccountOut}}  {{.Currency}} {{.AmountOut}}
  {{.AccountIn}}   {{.Currency}} {{.AmountIn}}

`

	recordType := formatRecord(record, config)

	// Create a new template and parse the letter into it.
	t := template.Must(template.New("transaction").Parse(recordTemplate))

	err := t.Execute(os.Stdout, recordType)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error executing template")
	}
}

// TODO: move this to internal package
func formatRecord(record []string, config Config) Record {
	var accountIn, accountOut, amountIn, amountOut, comment, currency, date, description, payee, raw string

	t, err := time.Parse(config.Csv.DateLayoutIn, record[config.Csv.Date])
	if err != nil {
		log.WithFields(log.Fields{
			"config.Csv.DateLayoutIn": config.Csv.DateLayoutIn,
			"record[config.Csv.Date]": record[config.Csv.Date],
			"error":                   err,
		}).Warn("error parsing date")
	}

	date = fmt.Sprintf(t.Format(config.Csv.DateLayoutOut))

	payee = record[config.Csv.Payee]
	currency = config.Csv.Currency
	description = record[config.Csv.Description]
	raw = fmt.Sprintf("%#v", record)

	var amount string

	if config.Csv.AmountIn != config.Csv.AmountOut {
		// explicit amountIn and amountOut fields
		if record[config.Csv.AmountIn] != "" {
			amount = formatAmount(record[config.Csv.AmountIn])
		} else if record[config.Csv.AmountOut] != "" {
			amount = fmt.Sprintf("-%s", record[config.Csv.AmountOut])
		}
	} else if config.Csv.AmountIn == config.Csv.AmountOut {
		// single amount field with signs to indicate transaction type
		amount = formatAmount(record[config.Csv.AmountIn])
	}

	// check the amount sign to determine the transaction type
	if regexp.MustCompile(`^-`).Match([]byte(amount)) {
		// it's a debit
		amountOut = amount
		amountIn = strings.ReplaceAll(amount, "-", "")
		accountOut = config.Csv.ProcessingAccount
		accountIn = config.Csv.DefaultAccount

		checkRules(config, payee, description, &accountIn, &comment)
	} else {
		// it's a credit
		amountIn = amount
		amountOut = fmt.Sprintf("-%s", amount)
		accountIn = config.Csv.ProcessingAccount
		accountOut = config.Csv.DefaultAccount

		checkRules(config, payee, description, &accountOut, &comment)
	}

	return Record{
		AccountIn:   accountIn,
		AccountOut:  accountOut,
		AmountIn:    amountIn,
		AmountOut:   amountOut,
		Comment:     comment,
		Currency:    currency,
		Date:        date,
		Description: description,
		Payee:       payee,
		Raw:         raw,
	}
}

// TODO: move this to internal package
func checkRules(config Config, payee, description string, account, comment *string) {
	for key := range config.TransactionsRules {
		log.WithFields(log.Fields{
			"description": description,
			"payee":       payee,
			"key":         key,
			"rule":        fmt.Sprintf("%#v", config.TransactionsRules[key]),
		}).Debug("iterating over rules")

		if checkRule(config.TransactionsRules[key]["match_payee"], payee) || checkRule(config.TransactionsRules[key]["match_description"], description) {
			applyRuleSetting(config.TransactionsRules[key]["set_account"], account)
			applyRuleSetting(config.TransactionsRules[key]["set_comment"], comment)
		}
	}
}

// TODO: move this to internal package
func applyRuleSetting(setting string, value *string) {
	if setting != "" {
		*value = setting
	}
}

// TODO: move this to internal package
func checkRule(expression, str string) bool {
	if expression == "" {
		// empty expressions will always match, so skip them
		return false
	}

	match := regexp.MustCompile(expression).FindString(str)

	log.WithFields(log.Fields{
		"expression": expression,
		"str":        str,
		"match":      match,
	}).Trace("checked rule")

	if match != "" {
		return true
	}

	// default
	return false
}

// TODO: move this to internal package
func formatAmount(val string) string {
	// comma as the decimal separator
	if regexp.MustCompile(`,\d{2}$`).Match([]byte(val)) {
		val = strings.ReplaceAll(val, ".", "")
		val = strings.ReplaceAll(val, ",", ".")
	}

	return val
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
}

// Typically this is in the root command, but since we don't actually
//  run the root command, that is have a Run property, then this wouldn't
//  as expected. So it's been moved here instead.
//
// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	// Set config defaults
	viper.SetDefault("csv.default_account", "Expenses:Unknown")
	viper.SetDefault("csv.processing_account", "Assets:Unknown")
	viper.SetDefault("csv.date_layout_out", "2006-01-02")
	viper.SetDefault("csv.separator", ";")
	viper.SetDefault("csv.skip", "0")

	viper.AutomaticEnv() // read in environment variables that match

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
