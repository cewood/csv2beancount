package internal

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config represents the config
type Config struct {
	Csv               CsvConfig
	TransactionsRules TransactionsRulesConfig
}

// TransactionsRulesConfig is a map of TransactionRule objects
type TransactionsRulesConfig map[string]TransactionRule

// TransactionRule is a set of values to match records with and update their values from
type TransactionRule struct {
	SetAccount       string
	SetComment       string
	MatchDescription string
	MatchPayee       string
}

// CsvConfig is the config for parsing the csv file
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

// Record represents a financial transaction record
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

// RecordTemplate is the default template for formatting records
const RecordTemplate = `{{.Date}} * "{{.Payee}}" "{{.Description}}"
  ; {{ .Raw }}
  {{.AccountOut}}  {{.AmountOut}} {{.Currency}}
  {{.AccountIn}}   {{.AmountIn}} {{.Currency}}

`

// GetTemplate ...
func GetTemplate(file string) string {
	if file == "" {
		return RecordTemplate
	}

	tpl, err := ioutil.ReadFile(file)
	if err != nil {
		log.WithFields(log.Fields{
			"file":  file,
			"error": err,
		}).Debug("error reading template file")

		return RecordTemplate
	}

	tplString := string(tpl)

	if _, err := template.New("tpl").Parse(tplString); err == nil {
		return tplString
	}

	return RecordTemplate
}

// getCsvReader ...
func getCsvReader(file io.Reader, skip int, sep rune, fields int) *csv.Reader {
	r := csv.NewReader(file)

	r.Comma = sep

	// Force this setting initially, after skipping any records it's
	//  updated with the user provided value or the default of 0.
	r.FieldsPerRecord = -1

	// Lines to skip at beginng of file, not including blank lines
	for skip > 0 {
		if record, err := r.Read(); err != nil {
			log.WithFields(log.Fields{
				"record": record,
				"error":  err,
			}).Trace("skipped line returned error")
		}
		skip = skip - 1
	}

	r.FieldsPerRecord = fields

	return r
}

// SetViperDefaults ...
func SetViperDefaults(cfgFile string) {
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
}

// ProcessCsvFile ...
func ProcessCsvFile(file io.Reader, config Config, template string) {
	r := getCsvReader(file, config.Csv.Skip, config.Csv.Separator, config.Csv.Fields)

L:
	for {
		record, err := r.Read()

		log.WithFields(log.Fields{
			"record": record,
			"error":  err,
		}).Trace("processing a csv record")

		// Check for errors
		switch err {
		case nil:
		case io.EOF:
			break L
		default:
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("error while reading csv file")
		}

		parseCsvRecord(record, config, template, os.Stdout)
	}
}

// GetConfig ...
func GetConfig() Config {
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

// getTransactionsRules ...
func getTransactionsRules(keys map[string]string) (rules TransactionsRulesConfig) {
	// rules = make(map[string]map[string]string)
	rules = make(TransactionsRulesConfig)

	for key := range keys {
		rules[key] = getTransactionRule(viper.GetStringMapString(fmt.Sprintf("transactions_rules.%s", key)))
	}

	return rules
}

func getTransactionRule(rule map[string]string) TransactionRule {
	return TransactionRule{
		SetAccount:       rule["set_account"],
		SetComment:       rule["set_comment"],
		MatchDescription: rule["match_description"],
		MatchPayee:       rule["match_payee"],
	}
}

// parseCsvRecord ...
func parseCsvRecord(record []string, config Config, tplString string, output io.Writer) {
	recordType := formatRecord(record, config)

	// Create a new template and parse the letter into it.
	t := template.Must(template.New("transaction").Parse(tplString))

	err := t.Execute(output, recordType)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error executing template")
	}
}

// formatRecord ...
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

	date = fmt.Sprint(t.Format(config.Csv.DateLayoutOut))

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

// checkRules ...
func checkRules(config Config, payee, description string, account, comment *string) {
	for key := range config.TransactionsRules {
		log.WithFields(log.Fields{
			"description": description,
			"payee":       payee,
			"key":         key,
			"rule":        fmt.Sprintf("%#v", config.TransactionsRules[key]),
		}).Debug("iterating over rules")

		if checkRule(config.TransactionsRules[key].MatchPayee, payee) || checkRule(config.TransactionsRules[key].MatchDescription, description) {
			applyRuleSetting(config.TransactionsRules[key].SetAccount, account)
			applyRuleSetting(config.TransactionsRules[key].SetComment, comment)
		}
	}
}

// applyRuleSetting ...
func applyRuleSetting(setting string, value *string) {
	if setting != "" {
		*value = setting
	}
}

// checkRule ...
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

	// Return bool indicating if match was not empty
	return match != ""
}

// formatAmount ...
func formatAmount(val string) string {
	// comma as the decimal separator
	if regexp.MustCompile(`,\d{2}$`).Match([]byte(val)) {
		val = strings.ReplaceAll(val, ".", "")
		val = strings.ReplaceAll(val, ",", ".")
	}

	return val
}
