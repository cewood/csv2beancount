package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CsvConfig is
type CsvConfig struct {
	AmountIn          int
	AmountOut         int
	Currency          string
	Date              int
	DateLayoutIn      string
	DateLayoutOut     string
	DefaultAccount    string // The default account for transactions if no rule matches
	Description       int
	Fields            int // Validate no. of fields; -1 is no check, 0 is infer from first row, and > 0 is explicit length
	Payee             int
	ProcessingAccount string // The account this export/CSV pertains to
	Separator         rune
	Skip              int
}

// Config represents the config
type Config struct {
	Csv               CsvConfig
	TransactionsRules map[string]map[string]string
}

// Record represents a financial transaction record
type Record struct {
	AccountIn   string
	AccountOut  string
	AmountIn    string
	AmountOut   string
	Comment     string
	Currency    string
	Date        string
	Description string
	Payee       string
	Raw         string
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

func getTransactionsRules(keys map[string]string) (rules map[string]map[string]string) {
	rules = make(map[string]map[string]string)

	for key := range keys {
		rules[key] = viper.GetStringMapString(fmt.Sprintf("transactions_rules.%s", key))
	}

	return rules
}

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
		if skip > 0 {
			skip = skip - 1
			// fmt.Println("skipping record, skip is:", skip)

			if skip == 0 {
				r.FieldsPerRecord = config.Csv.Fields
			}

			continue
		}

		// TODO : handle EOF error gracefully
		if err != nil {
			log.Fatal(err)
			break
		}

		parseCsvRecord(record, config)
	}
}

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

func formatRecord(record []string, config Config) Record {
	var accountIn, accountOut, amountIn, amountOut, comment, currency, date, description, payee, raw string

	// date = record[config.Csv.Date]
	t, err := time.Parse(config.Csv.DateLayoutIn, record[config.Csv.Date])
	if err != nil {
		fmt.Println("Error parsing date: ", err)
	}

	date = fmt.Sprintf(t.Format(config.Csv.DateLayoutOut))
	// date = record[config.Csv.Date]

	payee = record[config.Csv.Payee]
	currency = config.Csv.Currency
	description = record[config.Csv.Description]
	raw = fmt.Sprintf("%#v", record)

	// TODO : check for conf amountIn != amountOut
	//   in this case lets figure out if it's a debit or a credit
	//   and convert it to a formatted single value to have a single
	//   consistent code path below. So to summarise, if in and out
	//   aren't the same field, then convert the value to a positive
	//   or negative equivalent value and let the first block below
	//   handle all types of transactions for us

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

	// if config.Csv.AmountIn == config.Csv.AmountOut {
	// 	amount = formatAmount(record[config.Csv.AmountIn])

	// 	// we have to check the amount sign to determine the transaction type
	// 	if regexp.MustCompile(`^-`).Match([]byte(amount)) {
	// 		// it's a debit
	// 		amountOut = amount
	// 		amountIn = strings.ReplaceAll(amount, "-", "")
	// 		accountOut = config.Csv.ProcessingAccount
	// 		accountIn = config.Csv.DefaultAccount

	// 		checkRules(config, payee, description, &accountIn, &comment)
	// 	} else {
	// 		// it's a credit
	// 		amountIn = amount
	// 		amountOut = fmt.Sprintf("-%s", amount)

	// 		accountIn = config.Csv.ProcessingAccount
	// 		accountOut = config.Csv.DefaultAccount

	// 		checkRules(config, payee, description, &accountOut, &comment)
	// 	}
	// } else {
	// 	// handle explicity amountIn and amountOut case
	// 	if record[config.Csv.AmountIn] != "" {
	// 		// it's a credit
	// 		amountIn = formatAmount(record[config.Csv.AmountIn])
	// 		amountOut = fmt.Sprintf("-%s", record[config.Csv.AmountIn])

	// 		accountIn = config.Csv.ProcessingAccount
	// 		accountOut = config.Csv.DefaultAccount

	// 		checkRules(config, payee, description, &accountOut, &comment)
	// 	} else {
	// 		// it's a debit
	// 		amountOut = formatAmount(record[config.Csv.AmountOut])
	// 		amountIn = fmt.Sprintf("-%s", amountOut)
	// 		accountOut = config.Csv.ProcessingAccount
	// 		accountIn = config.Csv.DefaultAccount

	// 		checkRules(config, payee, description, &accountIn, &comment)
	// 	}
	// }

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

func applyRuleSetting(setting string, value *string) {
	if setting != "" {
		*value = setting
	}
}

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
