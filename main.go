package main

import (
	"fmt"
	"log"
	"os"
	"time"
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

	reportWorkers := len(config.Query)
	reportWorkersDone := make(chan interface{}, reportWorkers)

	// start work in n-amount of goroutines
	for i := 0; i < reportWorkers; i++ {
		go ReportRoutine(&config.Query[i], reportWorkersDone, input)
	}

	// wait for all goroutines to finish
	for i := 0; i < reportWorkers; i++ {
		<-reportWorkersDone
	}

	close(input)
	<-writerDone // wait for WriterRoutine to finish

	os.Exit(EXITCODE)
}
