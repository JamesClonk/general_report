package main

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-oci8" // TODO: adapt this here to your DB driver of choice!
)

func ReportRoutine(query *Query, done chan<- interface{}, writer chan<- WriterData) {
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

		procdone := make(chan interface{}, query.Range.Parallel)
		procinput := make(chan InputData, query.Range.Parallel)

		// start parallel processes
		for p := 0; p < query.Range.Parallel; p++ {
			go process(query, db, procdone, writer, procinput)
		}

		// feed input to processes
		for {
			if rangeEnd > rangeMax {
				rangeEnd = rangeMax
			}

			bindvars[query.Range.BindvarStart] = rangeStart
			bindvars[query.Range.BindvarEnd] = rangeEnd

			procinput <- InputData{bindvars}

			if rangeEnd >= rangeMax {
				break
			}
			rangeStart = rangeEnd + 1
			rangeEnd = rangeStart + query.Range.Stepsize - 1
		}
		close(procinput)

		// wait for all processes to finish
		for p := 0; p < query.Range.Parallel; p++ {
			<-procdone
		}

	} else {
		procdone := make(chan interface{}, 1)
		procinput := make(chan InputData, 1)
		go process(query, db, procdone, writer, procinput)
		procinput <- InputData{}
		close(procinput)
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

type InputData struct {
	Bindvars map[string]interface{}
}

func process(query *Query, db *sqlx.DB, done chan<- interface{}, writer chan<- WriterData, input <-chan InputData) {
	defer func() { done <- nil }()

	// printing out data requires a header, therefore we need to check if there is one provided already.
	// otherwise one will need to be constructed upon reading the first row
	var headerAvailable bool
	if len(query.Header) > 0 {
		headerAvailable = true
	}

	for in := range input {
		// prepare statement
		stmt, err := db.PrepareNamed(query.Statement)
		if err != nil {
			log.Println("could not prepare statement [%v]", query.Statement)
			log.Println(err)
			EXITCODE = 11
			continue
		}
		defer stmt.Close()

		// execute statement
		rows, err := stmt.Queryx(in.Bindvars)
		if err != nil {
			log.Printf("could not execute statement [%v], with bindvars: %v\n", query.Statement, in.Bindvars)
			log.Println(err)
			EXITCODE = 12
			continue
		}
		defer rows.Close()

		// go through resultset
		var csvLineCount, xlsLinecount int64
		for rows.Next() {
			data := make(map[string]interface{})
			err = rows.MapScan(data)
			if err != nil {
				log.Println(err)
				EXITCODE = 13
				continue
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

			// print out
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
}
