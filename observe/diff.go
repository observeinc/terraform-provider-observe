package observe

import (
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func diffSuppressDuration(k, old, new string, d *schema.ResourceData) bool {
	o, _ := time.ParseDuration(old)
	n, _ := time.ParseDuration(new)
	return o == n
}
