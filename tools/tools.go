//go:build tools

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "github.com/jstemmer/go-junit-report/v2"
	_ "gotest.tools/gotestsum"
)
