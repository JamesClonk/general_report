package main

import (
	"encoding/csv"
	"os"
)

type ScreenWriter struct {
	Out *csv.Writer
}

func (w ScreenWriter) Print(header []string, data map[string]interface{}) error {
	if err := w.Out.Write(convertToRecord(header, data)); err != nil {
		return err
	}
	return w.Flush()
}

func (w ScreenWriter) Close() error {
	return w.Flush()
}

func (w ScreenWriter) Flush() error {
	w.Out.Flush()
	return w.Out.Error()
}

func newScreenWriter(out *os.File, delimiter rune) *ScreenWriter {
	w := csv.NewWriter(out)
	w.Comma = delimiter
	return &ScreenWriter{w}
}

func newStdOutWriter(delimiter rune) *ScreenWriter {
	return newScreenWriter(os.Stdout, delimiter)
}

func newStdErrWriter(delimiter rune) *ScreenWriter {
	return newScreenWriter(os.Stderr, delimiter)
}
