package main

import (
	"encoding/xml"
	"io/ioutil"
	"strings"
	"unicode/utf8"
)

func loadConfigXML(filename string) (*Report, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Report
	err = xml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	for c, query := range config.Query {
		// setup delimiter
		if len(query.Delimiter) == 0 {
			query.Delimiter = ";"
		}
		delimiter, _ := utf8.DecodeRuneInString(query.Delimiter)

		// setup output writers
		if query.Output != nil {
			for _, screen := range query.Output.Screen {
				if strings.ToUpper(screen.Value) == "STDERR" {
					screen.Writer = newStdErrWriter(delimiter)
				} else {
					screen.Writer = newStdOutWriter(delimiter)
				}
			}
			for _, csv := range query.Output.Csv {
				csv.Value = replacePlaceholders(csv.Value)
				csv.Writer = newCsvWriter(csv.Value, delimiter)
			}
			for _, xls := range query.Output.Xls {
				xls.Value = replacePlaceholders(xls.Value) + ".csv" // add .csv since XLS(X) files are currently not yet supported
				xls.Writer = newXlsWriter(xls.Value, delimiter)
			}
		}

		// setup header
		if len(query.Header) > 0 {
			config.Query[c].Header = strings.Split(sanitizeHeader(query.Header[0]), string(delimiter))
		}

		// setup email
		if query.Email == nil {
			config.Query[c].Email = &Email{}
		}
	}

	return &config, nil
}

func sanitizeHeader(header string) string {
	return strings.Trim(header, "\t\n\r ")
}

func replacePlaceholders(filename string) string {
	filename = strings.Replace(filename, `{DATE}`, TIMESTAMP[:8], -1)
	filename = strings.Replace(filename, `{TIME}`, TIMESTAMP[8:], -1)
	filename = strings.Replace(filename, `{DATETIME}`, TIMESTAMP, -1)
	filename = strings.Replace(filename, `{TIMESTAMP}`, TIMESTAMP, -1)
	return filename
}

type Report struct {
	Query []Query `xml:"query"`
}

type Query struct {
	// all elements except Connection and Statement are optional
	Output     *Output    `xml:"output"`
	Email      *Email     `xml:"email"`
	Connection Connection `xml:"connection"`
	Range      *Range     `xml:"range"`
	Header     []string   `xml:"header"`
	Delimiter  string     `xml:"delimiter"`
	Statement  string     `xml:"statement"`
}

type Output struct {
	Csv    []*Csv    `xml:"csv"`
	Xls    []*Xls    `xml:"xls"`
	Screen []*Screen `xml:"screen"`
}

type Csv struct {
	Value     string `xml:",chardata"`
	Mail      bool   `xml:"mail,attr"`
	Temporary bool   `xml:"temporary,attr"`
	Writer    Writer
}

type Xls struct {
	Value      string `xml:",chardata"`
	Mail       bool   `xml:"mail,attr"`
	Temporary  bool   `xml:"temporary,attr"`
	Sheetname  string `xml:"sheetname,attr"`
	Autofilter bool   `xml:"autofilter,attr"`
	Writer     Writer
}

type Screen struct {
	Value  string `xml:",chardata"`
	Mail   bool   `xml:"mail,attr"`
	Writer Writer
}

type Email struct {
	SendEmptyReport bool     `xml:"send_empty_report,attr"`
	From            string   `xml:"from"`
	Subject         string   `xml:"subject"`
	Body            string   `xml:"body"`
	To              []string `xml:"to"`
	Cc              []string `xml:"cc"`
}

type Connection struct {
	Name     string `xml:"db_name"`
	User     string `xml:"db_user"`
	Password string `xml:"db_password"`
}

type Range struct {
	Start        int64  `xml:"start"`
	Stepsize     int64  `xml:"stepsize"`
	Steps        int64  `xml:"steps"`
	Parallel     int    `xml:"parallel"`
	BindvarStart string `xml:"bindvar_start"`
	BindvarEnd   string `xml:"bindvar_end"`
}
