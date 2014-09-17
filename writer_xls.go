package main

func newXlsWriter(filename string, delimiter rune) *CsvWriter {
	// XLS(X) writing is not supported for now. Fallback to CSV output..
	return newCsvWriter(filename, delimiter)
}
