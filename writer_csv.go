package main

import (
	"encoding/csv"
	"log"
	"os"
)

type CsvWriter struct {
	Out *csv.Writer
}

func (w CsvWriter) Print(header []string, data map[string]interface{}) error {
	return w.Out.Write(convertToRecord(header, data))
}

func (w CsvWriter) Close() error {
	return w.Flush()
}

func (w CsvWriter) Flush() error {
	w.Out.Flush()
	return w.Out.Error()
}

func newCsvWriter(filename string, delimiter rune) *CsvWriter {
	file, err := os.Create(filename)
	if err != nil {
		log.Println("ERROR: cannot create CSV output file: %v", filename)
		log.Fatalln(err)
	}

	w := csv.NewWriter(file)
	w.Comma = delimiter
	return &CsvWriter{w}
}
