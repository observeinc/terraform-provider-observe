package observe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

const maxOrderingAttempts = 3

func resourceIngestRouteOrder() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("ingest_route_order", "description"),
		CreateContext: resourceIngestRouteOrderCreate,
		ReadContext:   resourceIngestRouteOrderRead,
		UpdateContext: resourceIngestRouteOrderUpdate,
		DeleteContext: resourceIngestRouteOrderDelete,
		Importer:      &schema.ResourceImporter{StateContext: resourceIngestRouteOrderImport},
		Schema: map[string]*schema.Schema{
			"type":      {Type: schema.TypeString, Required: true, ForceNew: true, ValidateDiagFunc: validateStringInSlice(ingestRouteTypes, false), Description: describeEnums(ingestRouteTypes, descriptions.Get("ingest_route_order", "schema", "type"))},
			"route_ids": {Type: schema.TypeList, Required: true, MinItems: 1, Elem: &schema.Schema{Type: schema.TypeString, ValidateDiagFunc: validateID()}, Description: descriptions.Get("ingest_route_order", "schema", "route_ids")},
		},
	}
}

func resourceIngestRouteOrderCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	data.SetId(data.Get("type").(string))
	return resourceIngestRouteOrderReconcile(ctx, data, meta)
}

func resourceIngestRouteOrderRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	routeType := data.Get("type").(string)
	routes, err := meta.(*observeclient.Client).Rest.ListIngestRoutes(ctx, routeType)
	if err != nil {
		return diag.Errorf("failed to list ingest routes [type=%s]: %s", routeType, err)
	}
	configured := makeStrSlice(data.Get("route_ids").([]interface{}))
	if err := data.Set("route_ids", remoteRoutePriorityPrefix(configured, routes)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceIngestRouteOrderUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceIngestRouteOrderReconcile(ctx, data, meta)
}

func resourceIngestRouteOrderDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	data.SetId("")
	return nil
}

func resourceIngestRouteOrderReconcile(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*observeclient.Client).Rest
	routeType := data.Get("type").(string)
	configured := makeStrSlice(data.Get("route_ids").([]interface{}))
	for attempt := 0; attempt < maxOrderingAttempts; attempt++ {
		routes, err := client.ListIngestRoutes(ctx, routeType)
		if err != nil {
			return diag.Errorf("failed to list ingest routes: %s", err)
		}
		ordering, err := mergeIngestRouteOrdering(routeType, configured, routes)
		if err != nil {
			return diag.FromErr(err)
		}
		if _, err := client.UpdateIngestRoutePriorities(ctx, routeType, ordering); err == nil {
			return resourceIngestRouteOrderRead(ctx, data, meta)
		} else if !rest.HasStatusCode(err, http.StatusBadRequest) || attempt == maxOrderingAttempts-1 {
			return diag.Errorf("failed to update ingest route ordering: %s", err)
		}
	}
	return diag.Errorf("failed to update ingest route ordering after %d attempts", maxOrderingAttempts)
}

func resourceIngestRouteOrderImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	routeType := data.Id()
	if diags := validateStringInSlice(ingestRouteTypes, false)(routeType, nil); diags.HasError() {
		return nil, fmt.Errorf("invalid ingest route type %q", routeType)
	}
	if err := data.Set("type", routeType); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{data}, nil
}

func mergeIngestRouteOrdering(routeType string, configured []string, routes []rest.IngestRouteResource) ([]string, error) {
	routeByID := make(map[string]rest.IngestRouteResource, len(routes))
	for _, route := range routes {
		routeByID[route.Id] = route
	}
	seen := make(map[string]struct{}, len(configured))
	for _, id := range configured {
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("route_ids contains duplicate route ID %q", id)
		}
		seen[id] = struct{}{}
		route, ok := routeByID[id]
		if !ok {
			return nil, fmt.Errorf("route ID %q does not exist for this type", id)
		}
		if route.Type != "" && route.Type != routeType {
			return nil, fmt.Errorf("route ID %q has type %q, not %q", id, route.Type, routeType)
		}
		if route.ManagedBy != nil {
			return nil, fmt.Errorf("route ID %q is managed and cannot be ordered", id)
		}
		if route.Pipeline == "" {
			return nil, fmt.Errorf("route ID %q is the default route and cannot be ordered", id)
		}
	}

	ordering := append([]string(nil), configured...)
	for _, route := range routes {
		if route.ManagedBy == nil && route.Pipeline != "" {
			if _, included := seen[route.Id]; !included {
				ordering = append(ordering, route.Id)
				seen[route.Id] = struct{}{}
			}
		}
	}
	for _, route := range routes {
		if route.ManagedBy == nil && route.Pipeline == "" {
			if _, included := seen[route.Id]; included {
				continue
			}
			ordering = append(ordering, route.Id)
			seen[route.Id] = struct{}{}
		}
	}
	return ordering, nil
}

// remoteRoutePriorityPrefix returns the eligible remote prefix that Terraform manages.
// When no routes are configured, it returns every eligible route to support import.
func remoteRoutePriorityPrefix(configured []string, routes []rest.IngestRouteResource) []string {
	result := make([]string, 0, len(configured))
	for _, route := range routes {
		if route.ManagedBy == nil && route.Pipeline != "" {
			result = append(result, route.Id)
			if len(configured) > 0 && len(result) == len(configured) {
				break
			}
		}
	}
	return result
}
