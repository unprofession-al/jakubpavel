package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
)

var (
	configFile     string
	errorDirectory string
)

func init() {
	flag.StringVarP(&configFile, "config", "c", "config.yml", "name of the config file")
	flag.StringVarP(&errorDirectory, "error-reports", "e", "", "directory to write the error reports to")
}

func main() {
	flag.Parse()

	config, err := NewConfig(configFile)
	exitOnErr(err)

	checker, err := NewChecker(config.Checks)
	exitOnErr(err)

	results := checker.Run()
	for _, result := range results {
		fmt.Printf("%d,%s,%t,%v\n", result.Timestamp.UnixNano(), result.Name, result.OK(), result.RTT)
		if !result.OK() && errorDirectory != "" {
			writeErrorReport(result, errorDirectory)
		}
	}
}

func writeErrorReport(result CheckResult, path string) error {
	os.MkdirAll(path, os.ModePerm)
	filename := filepath.Join(path, fmt.Sprintf("%s-%d-error.report", result.Name, result.Timestamp.UnixNano()))
	data := []byte(result.String())
	return ioutil.WriteFile(filename, data, 0600)
}

// exitOnErr takes an arbitary number of errors and prints those to stderr
// if they are not nil. If any non-nil errors where passed the program will
// be exited.
func exitOnErr(errs ...error) {
	errNotNil := false
	for _, err := range errs {
		if err == nil {
			continue
		}
		errNotNil = true
		fmt.Fprintf(os.Stderr, "ERROR: %s", err.Error())
	}
	if errNotNil {
		fmt.Print("\n")
		os.Exit(-1)
	}
}
