package internal

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

var DefaultYamlConfig = `csv:
  amount_in: 7
  amount_out: 7
  currency: "EUR"
  date: 0
  date_layout_in: "02.01.2006"
  date_layout_out: "2006-01-02"
  default_account: "Expenses:Unknown"
  description: 4
  fields: 0
  payee: 2
  processing_account: "Assets:Unknown"
  separator: ;
  skip: 10
transactions_rules:
  BLAH:
    set_account: "set_account"
    set_comment: "set_comment"
    match_description: "match_description"
    match_payee: "match_payee"
`

var DefaultCsvConfig = CsvConfig{
	AmountIn:          0,
	AmountOut:         0,
	Currency:          "EUR",
	Date:              0,
	DateLayoutIn:      "",
	DateLayoutOut:     "",
	DefaultAccount:    "",
	Description:       0,
	Fields:            0,
	Payee:             0,
	ProcessingAccount: "",
	Separator:         []rune(";")[0],
	Skip:              0,
}

var DefaultConfigExample1 = Config{
	Csv: CsvConfig{
		AmountIn:          7,
		AmountOut:         7,
		Currency:          "EUR",
		Date:              0,
		DateLayoutIn:      "02.01.2006",
		DateLayoutOut:     "2006-01-02",
		DefaultAccount:    "Expenses:Unknown",
		Description:       4,
		Fields:            0,
		Payee:             2,
		ProcessingAccount: "Assets:Unknown",
		Separator:         ';',
		Skip:              10,
	},
	TransactionsRules: TransactionsRulesConfig{
		"blah": TransactionRule{
			SetAccount:       "set_account",
			SetComment:       "set_comment",
			MatchDescription: "match_description",
			MatchPayee:       "match_payee",
		},
	},
}

var DefaultCsvFile = `first_name,last_name,username
"Rob","Pike",rob
Ken,Thompson,ken
"Robert","Griesemer","gri"
`

var INGDiBaCsvFile = `Umsatzanzeige;Datei erstellt am: 28.03.2020 10:14
;Letztes Update: aktuell

IBAN;DE91 1000 0000 0123 4567 89
Kontoname;Cash
Bank;ING
Kunde;Joe Money
Zeitraum;01.04.2001 - 31.12.2000
Saldo;616,69;EUR

Sortierung;Datum absteigend

In der CSV-Datei finden Sie alle bereits gebuchten Ums<E4>tze. Die vorgemerkten Ums<E4>tze werden nicht aufgenommen, auch wenn sie in Ihrem Internetbanking angezeigt werden.

Buchung;Valuta;Auftraggeber/Empf<E4>nger;Buchungstext;Verwendungszweck;Saldo;W<E4>hrung;Betrag;W<E4>hrung
26.04.2019;26.04.2019;Acme Corp GmbH;Gehalt/Rente;LOHN / GEHALT 04/19;12.604,42;EUR;3.784,22;EUR
24.04.2019;29.04.2019;VISA RYANAIR;Lastschrift;NR8123456015 DUBLIN IE KAUFUMSATZ 18.04 223655 ARN74463669123456099978837;6.823,05;EUR;-16,00;EUR
24.04.2019;29.04.2019;VISA BLOCK HOUSE 1133;Lastschrift;NR8412345615 BERLIN KAUFUMSATZ 18.04 131250 ARN24463689108123456572752;6.839,05;EUR;-27,00;EUR
23.04.2019;26.04.2019;VISA CAR2GO DEUTSCHLAND GMB;Lastschrift;NR8412345615 LEINFELDEN- KAUFUMSATZ 17.04 211423 ARN74612345608000518071223;1.864,95;EUR;-12,22;EUR
23.04.2019;26.04.2019;VISA REWE MARKT GMBH-ZWNL O;Lastschrift;NR8412345615 BERLIN KAUFUMSATZ 17.04 211902 ARN74830729107123456039442;1.877,17;EUR;-6,58;EUR
23.04.2019;26.04.2019;VISA DUSSMANN D.KULTURKAUFH;Lastschrift;NR8412345615 BERLIN KAUFUMSATZ 16.04 ARN74830729107212345632429;1.883,75;EUR;-18,99;EUR
`

func TestGetCsvReader(t *testing.T) {
	var tests = []struct {
		name   string
		file   io.Reader
		skip   int
		sep    rune
		fields int
		err    error
		want   []string
	}{
		{
			"test #1 plain csv file",
			strings.NewReader(DefaultCsvFile),
			0,
			',',
			-1,
			nil,
			[]string{"first_name", "last_name", "username"},
		},
		{
			"test #2 ing-diba semicolon separated file",
			strings.NewReader(INGDiBaCsvFile),
			10,
			';',
			-1,
			nil,
			[]string{"Buchung", "Valuta", "Auftraggeber/Empf<E4>nger", "Buchungstext", "Verwendungszweck", "Saldo", "W<E4>hrung", "Betrag", "W<E4>hrung"},
		},
	}

	for _, tt := range tests {
		testname := tt.name
		t.Run(testname, func(t *testing.T) {
			r := getCsvReader(tt.file, tt.skip, tt.sep, tt.fields)
			record, err := r.Read()

			if !reflect.DeepEqual(record, tt.want) || err != tt.err {
				t.Errorf("got %v and %v, want %v and %v", record, err, tt.want, tt.err)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	var tests = []struct {
		name string
		conf string
		want Config
	}{
		{
			"test #1 standard yaml file",
			DefaultYamlConfig,
			DefaultConfigExample1,
		},
	}

	for _, tt := range tests {
		testname := tt.name
		t.Run(testname, func(t *testing.T) {
			// The "" is required, otherwise it expects the file specified
			SetViperDefaults("")

			err := viper.ReadConfig(strings.NewReader(tt.conf))
			if err != nil {
				fmt.Println("Error reading config: ", err)
			}
			cfg := GetConfig()

			if !reflect.DeepEqual(cfg, tt.want) {
				t.Errorf("got %v, want %v", cfg, tt.want)
			}
		})
	}
}

func TestCheckRules(t *testing.T) {
	var tests = []struct {
		name        string
		conf        Config
		payee       string
		desc        string
		account     string
		comment     string
		wantAccount string
		wantComment string
	}{
		{"test #1: match payee",
			Config{
				Csv: DefaultCsvConfig,
				TransactionsRules: TransactionsRulesConfig{
					"TEST": TransactionRule{
						SetAccount:       "updated_account",
						SetComment:       "updated_comment",
						MatchDescription: "",
						MatchPayee:       "payee",
					},
				},
			}, "payee", "some description", "default_account", "default_comment", "updated_account", "updated_comment",
		},
		{"test #2: match description",
			Config{
				Csv: DefaultCsvConfig,
				TransactionsRules: TransactionsRulesConfig{
					"TEST": TransactionRule{
						SetAccount:       "updated_account",
						SetComment:       "updated_comment",
						MatchDescription: "description",
						MatchPayee:       "",
					},
					"TEST2": TransactionRule{
						SetAccount:       "wont match",
						SetComment:       "",
						MatchDescription: "",
						MatchPayee:       "",
					},
				},
			}, "some payee", "description", "default_account", "default_comment", "updated_account", "updated_comment",
		},
	}

	for _, tt := range tests {
		testname := tt.name
		t.Run(testname, func(t *testing.T) {
			checkRules(tt.conf, tt.payee, tt.desc, &tt.account, &tt.comment)
			if tt.account != tt.wantAccount || tt.comment != tt.wantComment {
				t.Errorf("got %v and %v, wanted %v and %v", tt.account, tt.comment, tt.wantAccount, tt.wantComment)
			}
		})
	}
}

func TestApplyRuleSetting(t *testing.T) {
	var tests = []struct {
		setting string
		value   string
		want    string
	}{
		{"", "default", "default"},
		{"override", "default", "override"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("exp: '%s', str: '%s', want: '%v'", tt.setting, tt.value, tt.want)
		t.Run(testname, func(t *testing.T) {
			applyRuleSetting(tt.setting, &tt.value)
			if tt.value != tt.want {
				t.Errorf("got %v, want %v", tt.value, tt.want)
			}
		})
	}
}

func TestParseCsvRecord(t *testing.T) {
	var tests = []struct {
		name  string
		input []string
		want  string
	}{
		{
			"test #1 ING DiBa debit record",
			[]string{
				"24.04.2019",
				"29.04.2019",
				"VISA RYANAIR",
				"Lastschrift",
				"NR8123456015 DUBLIN IE KAUFUMSATZ 18.04 223655 ARN74463669123456099978837",
				"6.823,05",
				"EUR",
				"-16,00",
				"EUR",
			},
			`2019-04-24 * "VISA RYANAIR" "NR8123456015 DUBLIN IE KAUFUMSATZ 18.04 223655 ARN74463669123456099978837"
  ; []string{"24.04.2019", "29.04.2019", "VISA RYANAIR", "Lastschrift", "NR8123456015 DUBLIN IE KAUFUMSATZ 18.04 223655 ARN74463669123456099978837", "6.823,05", "EUR", "-16,00", "EUR"}
  Assets:Unknown  EUR -16.00
  Expenses:Unknown   EUR 16.00

`,
		},
		{
			"test #2 ING DiBa credit record",
			[]string{
				"26.04.2019",
				"26.04.2019",
				"Acme Corp GmbH",
				"Gehalt/Rente",
				"LOHN / GEHALT 04/19",
				"12.604,42",
				"EUR",
				"3.784,22",
				"EUR",
			},
			`2019-04-26 * "Acme Corp GmbH" "LOHN / GEHALT 04/19"
  ; []string{"26.04.2019", "26.04.2019", "Acme Corp GmbH", "Gehalt/Rente", "LOHN / GEHALT 04/19", "12.604,42", "EUR", "3.784,22", "EUR"}
  Expenses:Unknown  EUR -3784.22
  Assets:Unknown   EUR 3784.22

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			parseCsvRecord(tt.input, DefaultConfigExample1, RecordTemplate, buf)
			if buf.String() != tt.want {
				t.Errorf("got %v, want %v", buf.String(), tt.want)
			}
		})
	}
}

func TestCheckRule(t *testing.T) {
	var tests = []struct {
		exp  string
		str  string
		want bool
	}{
		{"match .*", "match this string", true},
		{"string .*", "no match", false},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("exp: '%s', str: '%s', want: '%v'", tt.exp, tt.str, tt.want)
		t.Run(testname, func(t *testing.T) {
			ans := checkRule(tt.exp, tt.str)
			if ans != tt.want {
				t.Errorf("got %v, want %v", ans, tt.want)
			}
		})
	}
}

func TestFormatAmount(t *testing.T) {
	var tests = []struct {
		input string
		want  string
	}{
		{"6,09", "6.09"},
		{"20,82", "20.82"},
		{"100,27", "100.27"},
		{"1.344,01", "1344.01"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("input: '%s', want: '%s'", tt.input, tt.want)
		t.Run(testname, func(t *testing.T) {
			ans := formatAmount(tt.input)
			if ans != tt.want {
				t.Errorf("got '%s', want '%s'", ans, tt.want)
			}
		})
	}
}
