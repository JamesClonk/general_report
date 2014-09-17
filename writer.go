package main

import (
	"fmt"
	"log"
)

type Writer interface {
	Print([]string, map[string]interface{}) error
	Close() error
	Flush() error
}

type Record []string

func convertToRecord(header []string, data map[string]interface{}) (record Record) {
	for _, key := range header {
		record = append(record, fmt.Sprint(data[key]))
	}
	return
}

type WriterData struct {
	Writer Writer
	Header []string
	Data   map[string]interface{}
	Flush  bool
}

func WriterRoutine(done chan<- interface{}, input <-chan WriterData) {
	defer func() { done <- nil }()

	for in := range input {
		if err := in.Writer.Print(in.Header, in.Data); err != nil {
			log.Println(err)
			EXITCODE = 5
		}

		if in.Flush {
			if err := in.Writer.Flush(); err != nil {
				log.Println(err)
				EXITCODE = 6
			}
		}
	}
}

func flushFiles(query *Query) {
	if query.Output != nil {
		for _, csv := range query.Output.Csv {
			csv.Writer.Flush()
		}
		for _, xls := range query.Output.Xls {
			xls.Writer.Flush()
		}
	}
}

func closeFiles(query *Query) {
	if query.Output != nil {
		for _, csv := range query.Output.Csv {
			csv.Writer.Close()
		}
		for _, xls := range query.Output.Xls {
			xls.Writer.Close()
		}
	}
}
