package observe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
	gql "github.com/observeinc/terraform-provider-observe/client/meta"
	"github.com/observeinc/terraform-provider-observe/client/oid"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

// the resource name is "resource_grants" (since it manages grants for another resource)
func resourceResourceGrants() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("resource_grants", "description"),
		CreateContext: resourceResourceGrantsCreate,
		ReadContext:   resourceResourceGrantsRead,
		UpdateContext: resourceResourceGrantsUpdate,
		DeleteContext: resourceResourceGrantsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: resourceResourceGrantsCustomizeDiff,
		Schema: map[string]*schema.Schema{
			"oid": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(validRbacV2Types...),
				DiffSuppressFunc: diffSuppressOIDVersion,
				Description:      descriptions.Get("resource_grants", "schema", "oid"),
			},
			"grant": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     resourceGrantsGrant(),
			},
		},
	}
}

func resourceGrantsGrant() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"subject": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateOID(oid.TypeUser, oid.TypeRbacGroup),
				Description:      descriptions.Get("resource_grants", "schema", "grant", "subject"),
			},
			"role": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateEnums(validGrantRoles),
				Description:      descriptions.Get("resource_grants", "schema", "grant", "role"),
			},
			"oid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceResourceGrantsCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	// ForceNew if the oid changed, excluding dataset version changes.
	// (We can't use the normal ForceNew since even with the diff suppress, it still
	// marks the resource for replacement when the dataset version changes)
	if d.HasChange("oid") {
		oldVal, newVal := d.GetChange("oid")
		oldOid, oldErr := oid.NewOID(oldVal.(string))
		newOid, newErr := oid.NewOID(newVal.(string))
		if oldErr == nil && newErr == nil {
			if oldOid.Type != newOid.Type || oldOid.Id != newOid.Id {
				d.ForceNew("oid")
			}
		}
	}
	return nil
}

func resourceResourceGrantsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	resourceOid, err := oid.NewOID(d.Get("oid").(string))
	if err != nil {
		return diag.Errorf("error parsing resource oid: %s", err.Error())
	}

	existingStatements, err := client.GetRbacResourceStatements(ctx, []string{resourceOid.Id})
	if err != nil {
		return diag.Errorf("failed to get existing grants: %s", err.Error())
	}

	existingGrantsSet := grantsToSet(existingStatements)
	newGrantsSet := d.Get("grant").(*schema.Set)
	diags := mutateResourceGrants(ctx, client, resourceOid, existingGrantsSet, newGrantsSet)
	if diags.HasError() {
		return diags
	}

	// Because the terraform provider sdk uses this special ID field to determine whether
	// the resource already exists, we must set it to some value despite there not really
	// being an actual object corresponding to the resource.
	// Setting it to the resource OID lets us use only this ID in resourceGrantsRead,
	// which has the benefit of allowing ImportStatePassthroughContext to just work.
	d.SetId(resourceOid.String())
	return resourceResourceGrantsRead(ctx, d, m)
}

func resourceResourceGrantsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	// Using d.Id() instead of the oid field allows ImportStatePassthroughContext to work.
	// (when importing, only the id is defined)
	resourceOid, err := oid.NewOID(d.Id())
	if err != nil {
		return diag.Errorf("error parsing id: %s", err.Error())
	}

	grants, err := client.GetRbacResourceStatements(ctx, []string{resourceOid.Id})
	if err != nil {
		return diag.Errorf("failed to get grants: %s", err.Error())
	}

	return grantsToResourceData(*resourceOid, grants, d)
}

func resourceResourceGrantsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	resourceOid, err := oid.NewOID(d.Get("oid").(string))
	if err != nil {
		return diag.Errorf("error parsing resource oid: %s", err.Error())
	}

	old, new := d.GetChange("grant")
	return mutateResourceGrants(ctx, client, resourceOid, old.(*schema.Set), new.(*schema.Set))
}

func resourceResourceGrantsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*observe.Client)

	resourceOid, err := oid.NewOID(d.Get("oid").(string))
	if err != nil {
		return diag.Errorf("error parsing resource oid: %s", err.Error())
	}

	existingGrants := d.Get("grant").(*schema.Set)
	emptySet := schema.NewSet(schema.HashResource(resourceGrantsGrant()), []interface{}{})
	return mutateResourceGrants(ctx, client, resourceOid, existingGrants, emptySet)
}

func mutateResourceGrants(ctx context.Context, client *observe.Client, resourceOid *oid.OID, oldGrants *schema.Set, newGrants *schema.Set) diag.Diagnostics {
	createGrants := newGrants.Difference(oldGrants)
	deleteGrants := oldGrants.Difference(newGrants)

	toCreate, diags := newGrantsInput(resourceOid, createGrants)
	if diags.HasError() {
		return diags
	}
	var toDelete []string
	for _, g := range deleteGrants.List() {
		grantOid, err := oid.NewOID(g.(map[string]interface{})["oid"].(string))
		if err != nil {
			return diag.Errorf("error parsing grant oid: %s", err.Error())
		}
		toDelete = append(toDelete, grantOid.Id)
	}

	_, err := client.MutateRbacStatements(ctx, toCreate, nil, toDelete)
	if err != nil {
		return diag.Errorf("failed to update grants: %s", err.Error())
	}
	return nil
}

// for now, translates grants into rbac statements until api support for grants is added
func newGrantsInput(resourceOid *oid.OID, grants *schema.Set) (input []gql.RbacStatementInput, diags diag.Diagnostics) {
	for _, g := range grants.List() {
		subject, err := newGrantSubjectInput(g.(map[string]interface{})["subject"].(string))
		if err != nil {
			return nil, diag.FromErr(err)
		}
		grantRole := GrantRole(toCamel(g.(map[string]interface{})["role"].(string)))
		role, err := grantRole.ToRbacRole()
		if err != nil {
			return nil, diag.FromErr(err)
		}
		object, err := newGrantObjectInput(grantRole, &resourceOid.Id)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		input = append(input, gql.RbacStatementInput{
			Subject: subject,
			Role:    role,
			Object:  object,
			Version: intPtr(2),
		})
	}
	return
}

// for now, receives an rbac statement type and translates it until api support for grants is added
func grantsToResourceData(oid oid.OID, grants []gql.RbacStatement, data *schema.ResourceData) (diags diag.Diagnostics) {
	if err := data.Set("oid", oid.String()); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := data.Set("grant", grantsToSet(grants)); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}

func grantsToSet(grants []gql.RbacStatement) *schema.Set {
	grantSet := schema.NewSet(schema.HashResource(resourceGrantsGrant()), []interface{}{})
	for _, g := range grants {
		grant := make(map[string]interface{})
		grant["subject"] = flattenGrantSubject(g.Subject)
		role, _ := flattenRoleAndObject(g.Role, g.Object)
		grant["role"] = role
		grant["oid"] = g.Oid().String()
		grantSet.Add(grant)
	}
	return grantSet
}
