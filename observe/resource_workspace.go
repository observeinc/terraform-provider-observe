package observe

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceWorkspace() *schema.Resource {
	return &schema.Resource{
		Create: resourceWorkspaceCreate,
		Read:   resourceWorkspaceRead,
		Delete: resourceWorkspaceDelete,

		Schema: map[string]*schema.Schema{},
	}
}

func resourceWorkspaceCreate(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("not yet implemented")
}

func resourceWorkspaceRead(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("not yet implemented")
}

func resourceWorkspaceUpdate(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("not yet implemented")
}

func resourceWorkspaceDelete(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("not yet implemented")
}
