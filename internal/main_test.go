package internal

import (
	"fmt"
	"testing"
)

func TestCheckRules(t *testing.T) {
	// TODO: add name to test struct for easier identification
	var tests = []struct {
		conf         Config
		payee        string
		desc         string
		account      string
		comment      string
		want_account string
		want_comment string
	}{
		{Config{
			Csv: CsvConfig{
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
			},
			TransactionsRules: map[string]map[string]string{
				"CSOL": map[string]string{
					set_account:       "updated_account",
					set_comment:       "",
					match_description: "description",
					match_payee:       "payee",
				},
			}, "payee", "description", "default_account", "default_comment", "default_account", "default_comment"},
		},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("exp: '%s', str: '%s', want: '%v'", tt.setting, tt.value, tt.want)
		t.Run(testname, func(t *testing.T) {
			checkRules(tt.conf, &tt.payee, tt.desc, &tt.account, &tt.comment)
			if tt.account != tt.want_account || tt.comment != tt.want_comment {
				t.Errorf("got %v and %v, wanted %v and %v", tt.account, tt.comment, tt.want_account, tt.want_comment)
			}
		})
	}
}

func TestApplyRuleSetting(t *testing.T) {
	// TODO: add name to test struct for easier identification
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
