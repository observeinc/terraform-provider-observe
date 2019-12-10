package observe

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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
	panic("not yet implemented")
	return nil
}

func resourceWorkspaceRead(d *schema.ResourceData, meta interface{}) error {
	panic("not yet implemented")
	return nil
}

func resourceWorkspaceUpdate(d *schema.ResourceData, meta interface{}) error {
	panic("not yet implemented")
	return nil
}

func resourceWorkspaceDelete(d *schema.ResourceData, meta interface{}) error {
	panic("not yet implemented")
	return nil
}
