package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-oci8"
)

var (
	EXITCODE  = 0
	TIMESTAMP = time.Now().Format(`20060102150405`)
	STDOUT    = newStdOutWriter(';') // default output to use in case nothing else is defined in config xml
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %v <xml-config-file>\n", os.Args[0])
		os.Exit(1)
	}

	os.Setenv("NLS_LANG", "")

	config, err := loadConfigXML(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	writerDone := make(chan interface{}, 1)
	input := make(chan WriterData, 100)
	go WriterRoutine(writerDone, input)

	workers := len(config.Query)
	workersDone := make(chan interface{}, workers)

	// start work in n-amount of goroutines
	for i := 0; i < workers; i++ {
		go report(&config.Query[i], workersDone, input)
	}

	// wait for all goroutines to finish
	for i := 0; i < workers; i++ {
		<-workersDone
	}

	close(input)
	<-writerDone // wait for WriterRoutine to finish

	os.Exit(EXITCODE)
}

func report(query *Query, done chan<- interface{}, writer chan<- WriterData) {
	defer func() { done <- nil }()

	connstr := query.Connection.User + "/" + query.Connection.Password + "@" + query.Connection.Name // TODO: adapt this here to your DB driver of choice!
	db, err := sqlx.Connect("oci8", connstr)                                                         // TODO: adapt this here to your DB driver of choice!
	if err != nil {
		log.Println("could not connect to [%v]", connstr)
		log.Println(err)
		EXITCODE = 10
		return
	}
	defer db.Close()

	// run queries in parallel if defined
	var bindvars map[string]interface{} = make(map[string]interface{})
	if query.Range != nil {
		rangeStart := query.Range.Start
		rangeEnd := rangeStart + query.Range.Stepsize - 1
		rangeMax := rangeStart + (query.Range.Stepsize * query.Range.Steps)

		var procs int
		procdone := make(chan interface{}, query.Range.Parallel)
		for {
			// iterate over RANGE
			for ; procs < query.Range.Parallel && rangeEnd <= rangeMax; procs++ {
				bindvars[query.Range.BindvarStart] = rangeStart
				bindvars[query.Range.BindvarEnd] = rangeEnd

				go process(query, db, bindvars, procdone, writer)

				if rangeEnd == rangeMax {
					break
				}

				rangeStart = rangeEnd + 1
				rangeEnd = rangeStart + query.Range.Stepsize - 1
				if rangeEnd > rangeMax {
					rangeEnd = rangeMax
				}
			}
			<-procdone
			procs--

			if rangeEnd >= rangeMax {
				for ; procs > 0; procs-- {
					<-procdone
				}
				break
			}
		}
	} else {
		procdone := make(chan interface{}, 1)
		go process(query, db, bindvars, procdone, writer)
		<-procdone
	}

	flushFiles(query)
	closeFiles(query)

	sendEmails(query)

	// delete temporary files
	if query.Output != nil {
		for _, csv := range query.Output.Csv {
			if csv.Temporary {
				if err := os.Remove(csv.Value); err != nil {
					log.Println(err)
					EXITCODE = 7
				}
			}
		}
		for _, xls := range query.Output.Xls {
			if xls.Temporary {
				if err := os.Remove(xls.Value); err != nil {
					log.Println(err)
					EXITCODE = 8
				}
			}
		}
	}
}

func process(query *Query, db *sqlx.DB, bindvars map[string]interface{}, done chan<- interface{}, writer chan<- WriterData) {
	defer func() { done <- nil }()

	// prepare statement
	stmt, err := db.PrepareNamed(query.Statement)
	if err != nil {
		log.Println("could not prepare statement [%v]", query.Statement)
		log.Println(err)
		EXITCODE = 11
		return
	}
	defer stmt.Close()

	rows, err := stmt.Queryx(bindvars)
	if err != nil {
		log.Printf("could not execute statement [%v], with bindvars: %v\n", query.Statement, bindvars)
		log.Println(err)
		EXITCODE = 12
		return
	}
	defer rows.Close()

	// printing out data requires a header, therefore we need to check if there is one provided already.
	// otherwise one will need to be constructed upon reading the first row
	var headerAvailable bool
	if len(query.Header) > 0 {
		headerAvailable = true
	}

	var csvLineCount, xlsLinecount int64
	for rows.Next() {
		data := make(map[string]interface{})
		err = rows.MapScan(data)
		if err != nil {
			log.Println(err)
			EXITCODE = 13
		}

		// setup header if not yet provided
		if !headerAvailable {
			header, err := rows.Columns()
			if err != nil {
				log.Println(err)
				EXITCODE = 14
			}
			query.Header = header
			headerAvailable = true
		}

		if query.Output != nil {
			for _, screen := range query.Output.Screen {
				writer <- WriterData{screen.Writer, query.Header, data, false}
			}
			for _, csv := range query.Output.Csv {
				writer <- WriterData{csv.Writer, query.Header, data, csvLineCount >= 10}
				if csvLineCount >= 10 {
					csvLineCount = 0
				}
				csvLineCount++
			}
			for _, xls := range query.Output.Xls {
				writer <- WriterData{xls.Writer, query.Header, data, xlsLinecount >= 10}
				if xlsLinecount >= 10 {
					xlsLinecount = 0
				}
				xlsLinecount++
			}
		} else {
			writer <- WriterData{STDOUT, query.Header, data, false}
		}
	}
}
