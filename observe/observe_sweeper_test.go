package observe

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/acctest"
)

func init() {
	acctest.UseBinaryDriver("observe", Provider)
}
