package client

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"testing"
)

var update = flag.Bool("update", false, "update result files")

func readJSONFile(tb testing.TB, filename string, v interface{}) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		tb.Fatalf("failed to read file: %s", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		tb.Fatalf("failed to parse json for %q: %s", filename, err)
	}
	return
}

func writeJSONFile(tb testing.TB, filename string, v interface{}) {
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		tb.Fatal(err)
	}
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		tb.Fatal(err)
	}
}
