package meta

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// datasetSearchSelection matches a datasetSearch field selection, aliased or not.
var datasetSearchSelection = regexp.MustCompile(`(^|[\s{:])datasetSearch\b`)

// The provider must not query the legacy datasetSearch root. The schema mirror
// may still declare it.
func TestNoDatasetSearchOperation(t *testing.T) {
	files, err := filepath.Glob("../internal/meta/operation/*.graphql")
	if err != nil || len(files) == 0 {
		t.Fatalf("no operation files found: %v", err)
	}
	files = append(files, "genqlient.generated.go")
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if loc := datasetSearchSelection.FindIndex(src); loc != nil {
			t.Errorf("%s selects datasetSearch at byte %d", f, loc[0])
		}
	}
}
