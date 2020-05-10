# csv2beancount

A small utility to convert your csv file of bank transactions to beancount format.


## Example

An example configuration file `config.yaml`:

```yaml
csv:
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
  skip: 11
transactions_rules:
  ACME:
    match_payee: "Acme Corp GmbH"
    set_account: "Income:Salary:AcmeCorp"
    set_comment: "Salary from Acme Corp GmbH"
```


an example input file `data.csv`:

```
Umsatzanzeige;Datei erstellt am: 28.03.2020 10:14
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
```


and an example of invoking csv2beancount:

```shell
$ csv2beancount convert --config examples/example_ing-diba.yaml examples/example_ing-diba.csv

2019-04-26 * "Acme Corp GmbH" "LOHN / GEHALT 04/19"
  ; []string{"26.04.2019", "26.04.2019", "Acme Corp GmbH", "Gehalt/Rente", "LOHN / GEHALT 04/19", "12.604,42", "EUR", "3.784,22", "EUR"}
  Income:Salary:AcmeCorp  EUR -3784.22
  Assets:Unknown   EUR 3784.22

2019-04-24 * "VISA RYANAIR" "NR8123456015 DUBLIN IE KAUFUMSATZ 18.04 223655 ARN74463669123456099978837"
  ; []string{"24.04.2019", "29.04.2019", "VISA RYANAIR", "Lastschrift", "NR8123456015 DUBLIN IE KAUFUMSATZ 18.04 223655 ARN74463669123456099978837", "6.823,05", "EUR", "-16,00", "EUR"}
  Assets:Unknown  EUR -16.00
  Expenses:Unknown   EUR 16.00

2019-04-24 * "VISA BLOCK HOUSE 1133" "NR8412345615 BERLIN KAUFUMSATZ 18.04 131250 ARN24463689108123456572752"
  ; []string{"24.04.2019", "29.04.2019", "VISA BLOCK HOUSE 1133", "Lastschrift", "NR8412345615 BERLIN KAUFUMSATZ 18.04 131250 ARN24463689108123456572752", "6.839,05", "EUR", "-27,00", "EUR"}
  Assets:Unknown  EUR -27.00
  Expenses:Unknown   EUR 27.00

2019-04-23 * "VISA CAR2GO DEUTSCHLAND GMB" "NR8412345615 LEINFELDEN- KAUFUMSATZ 17.04 211423 ARN74612345608000518071223"
  ; []string{"23.04.2019", "26.04.2019", "VISA CAR2GO DEUTSCHLAND GMB", "Lastschrift", "NR8412345615 LEINFELDEN- KAUFUMSATZ 17.04 211423 ARN74612345608000518071223", "1.864,95", "EUR", "-12,22", "EUR"}
  Assets:Unknown  EUR -12.22
  Expenses:Unknown   EUR 12.22

2019-04-23 * "VISA REWE MARKT GMBH-ZWNL O" "NR8412345615 BERLIN KAUFUMSATZ 17.04 211902 ARN74830729107123456039442"
  ; []string{"23.04.2019", "26.04.2019", "VISA REWE MARKT GMBH-ZWNL O", "Lastschrift", "NR8412345615 BERLIN KAUFUMSATZ 17.04 211902 ARN74830729107123456039442", "1.877,17", "EUR", "-6,58", "EUR"}
  Assets:Unknown  EUR -6.58
  Expenses:Unknown   EUR 6.58

2019-04-23 * "VISA DUSSMANN D.KULTURKAUFH" "NR8412345615 BERLIN KAUFUMSATZ 16.04 ARN74830729107212345632429"
  ; []string{"23.04.2019", "26.04.2019", "VISA DUSSMANN D.KULTURKAUFH", "Lastschrift", "NR8412345615 BERLIN KAUFUMSATZ 16.04 ARN74830729107212345632429", "1.883,75", "EUR", "-18,99", "EUR"}
  Assets:Unknown  EUR -18.99
  Expenses:Unknown   EUR 18.99
```


## Configuration Syntax

```yaml
csv:
  amount_in: 7  # The index of this field in the csv file, zero indexed
  amount_out: 7  # The index of this field in the csv file, zero indexed
  currency: "EUR"
  date: 0  # The index of this field in the csv file, zero indexed
  date_layout_in: "02.01.2006"  # The date format of the csv file, expressed in Go [Time.Format](https://golang.org/pkg/time/#pkg-constants)
  date_layout_out: "2006-01-02"  # The date format to use for output, expressed in Go [Time.Format](https://golang.org/pkg/time/#pkg-constants)
  default_account: "Expenses:Unknown"  # The default account for transactions if no rule matches
  description: 4  # The index of this field in the csv file, zero indexed
  fields: 0  # Whether to validate no. of fields; -1 is no check, 0 is infer from first row, and > 0 is explicit length
  payee: 2  # The index of this field in the csv file, zero indexed
  processing_account: "Assets:ING-DiBa:Account"  # The account this export/CSV pertains to
  separator: ;  # The field separator for the csv file, per the [encoding/csv/#Reader](https://golang.org/pkg/encoding/csv/#Reader) type
  skip: 11  # The number of lines to skip, not including blank lines which are excluded already by Go
transactions_rules:
  ACME:  # This is just a key to identify a rule, it can be anything you like
    set_account: "Income:Salary:AcmeCorp"  # The account to use for the other side of this transaction
    set_comment: "Salary from Acme Corp GmbH"  # The comment to add for this record, optional
    match_description: "LOHN / GEHALT"  # Any valid [RE2 expression](https://github.com/google/re2/wiki/Syntax)
    match_payee: "Acme Corp GmbH"  # Any valid [RE2 expression](https://github.com/google/re2/wiki/Syntax)
```


## Why another csv2beancount

This tool is heavily influenced by [PaNaVTEC/csv2beancount](https://github.com/PaNaVTEC/csv2beancount) and [alexkursell/rust-csv2beancount](https://github.com/alexkursell/rust-csv2beancount).

The motivation for creating another implementation was to add the following functionality:

 1. Custom templates

 1. Matching on description as well as payee

 1. Stronger support for regular expressions

 1. Stricter date formatting

