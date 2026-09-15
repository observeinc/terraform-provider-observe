package observe

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	observeclient "github.com/observeinc/terraform-provider-observe/client"
	"github.com/observeinc/terraform-provider-observe/client/rest"
	"github.com/observeinc/terraform-provider-observe/observe/descriptions"
)

var ingestRouteTypes = []string{"otellogs", "otelmetrics", "oteltraces", "prometheus", "k8sentity", "any"}

func resourceIngestRoute() *schema.Resource {
	return &schema.Resource{
		Description:   descriptions.Get("ingest_route", "description"),
		CreateContext: resourceIngestRouteCreate,
		ReadContext:   resourceIngestRouteRead,
		UpdateContext: resourceIngestRouteUpdate,
		DeleteContext: resourceIngestRouteDelete,
		Importer:      &schema.ResourceImporter{StateContext: resourceIngestRouteImport},
		Schema: map[string]*schema.Schema{
			"route_id":                 {Type: schema.TypeString, Computed: true, Description: descriptions.Get("ingest_route", "schema", "route_id")},
			"type":                     {Type: schema.TypeString, Required: true, ForceNew: true, ValidateDiagFunc: validateStringInSlice(ingestRouteTypes, false), Description: describeEnums(ingestRouteTypes, descriptions.Get("ingest_route", "schema", "type"))},
			"pipeline":                 {Type: schema.TypeString, Required: true, ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace), Description: descriptions.Get("ingest_route", "schema", "pipeline")},
			"destination_id":           {Type: schema.TypeString, Required: true, ValidateDiagFunc: validateID(), Description: descriptions.Get("ingest_route", "schema", "destination_id")},
			"secondary_destination_id": {Type: schema.TypeString, Optional: true, ValidateDiagFunc: validateID(), Description: descriptions.Get("ingest_route", "schema", "secondary_destination_id")},
			"enabled":                  {Type: schema.TypeBool, Optional: true, Default: true, Description: descriptions.Get("ingest_route", "schema", "enabled")},
		},
	}
}

func resourceIngestRouteCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	req := ingestRouteRequestFromResourceData(data)
	routeType := data.Get("type").(string)
	route, err := meta.(*observeclient.Client).Rest.CreateIngestRoute(ctx, routeType, req)
	if err != nil {
		return diag.Errorf("failed to create ingest route: %s", err)
	}
	data.SetId(ingestRouteStateID(routeType, route.Id))
	if data.Get("enabled").(bool) {
		enabled := true
		if _, err := meta.(*observeclient.Client).Rest.UpdateIngestRoute(ctx, routeType, route.Id, &rest.IngestRouteUpdateRequest{Enabled: &enabled}); err != nil {
			return diag.Errorf("failed to enable created ingest route [type=%s, id=%s]: %s", routeType, route.Id, err)
		}
	}
	return resourceIngestRouteRead(ctx, data, meta)
}

func resourceIngestRouteRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	routeType, routeID, err := parseIngestRouteImportID(data.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	route, err := meta.(*observeclient.Client).Rest.GetIngestRoute(ctx, routeType, routeID)
	if err != nil {
		if rest.HasStatusCode(err, http.StatusNotFound) {
			data.SetId("")
			return nil
		}
		return diag.Errorf("failed to retrieve ingest route [type=%s, id=%s]: %s", routeType, routeID, err)
	}
	if ingestRouteIsUnmanageable(route) {
		data.SetId("")
		return nil
	}
	return ingestRouteToResourceData(data, routeType, route)
}

func resourceIngestRouteUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	routeType, routeID, err := parseIngestRouteImportID(data.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	req := &rest.IngestRouteUpdateRequest{Pipeline: stringPtr(data.Get("pipeline").(string)), DestinationId: stringPtr(data.Get("destination_id").(string)), Enabled: boolPtr(data.Get("enabled").(bool)), SecondaryDestinationIdSpecified: true}
	if secondaryDestinationID, ok := data.GetOk("secondary_destination_id"); ok {
		req.SecondaryDestinationId = stringPtr(secondaryDestinationID.(string))
	}
	if _, err := meta.(*observeclient.Client).Rest.UpdateIngestRoute(ctx, routeType, routeID, req); err != nil {
		return diag.Errorf("failed to update ingest route [type=%s, id=%s]: %s", routeType, routeID, err)
	}
	return resourceIngestRouteRead(ctx, data, meta)
}

func resourceIngestRouteDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	routeType, routeID, err := parseIngestRouteImportID(data.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := meta.(*observeclient.Client).Rest.DeleteIngestRoute(ctx, routeType, routeID); err != nil && !rest.HasStatusCode(err, http.StatusNotFound) {
		return diag.Errorf("failed to delete ingest route [type=%s, id=%s]: %s", routeType, routeID, err)
	}
	data.SetId("")
	return nil
}

func ingestRouteRequestFromResourceData(data *schema.ResourceData) *rest.IngestRouteCreateRequest {
	req := &rest.IngestRouteCreateRequest{Pipeline: data.Get("pipeline").(string), DestinationId: data.Get("destination_id").(string)}
	if secondaryDestinationID, ok := data.GetOk("secondary_destination_id"); ok {
		req.SecondaryDestinationId = stringPtr(secondaryDestinationID.(string))
	}
	return req
}

func ingestRouteToResourceData(data *schema.ResourceData, routeType string, route *rest.IngestRouteResource) (diags diag.Diagnostics) {
	set := func(key string, value interface{}) {
		if err := data.Set(key, value); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}
	data.SetId(ingestRouteStateID(routeType, route.Id))
	set("route_id", route.Id)
	set("type", routeType)
	set("pipeline", route.Pipeline)
	set("destination_id", route.DestinationId)
	set("secondary_destination_id", route.SecondaryDestinationId)
	set("enabled", route.Enabled)
	return diags
}

func resourceIngestRouteImport(ctx context.Context, data *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	routeType, routeID, err := parseIngestRouteImportID(data.Id())
	if err != nil {
		return nil, err
	}
	route, err := meta.(*observeclient.Client).Rest.GetIngestRoute(ctx, routeType, routeID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ingest route [type=%s, id=%s]: %w", routeType, routeID, err)
	}
	if ingestRouteIsUnmanageable(route) {
		return nil, fmt.Errorf("ingest route [type=%s, id=%s] is default or managed and cannot be imported", routeType, routeID)
	}
	if err := data.Set("type", routeType); err != nil {
		return nil, err
	}
	data.SetId(ingestRouteStateID(routeType, routeID))
	return []*schema.ResourceData{data}, nil
}

func parseIngestRouteImportID(value string) (string, string, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected ingest route import ID in type/id format")
	}
	if diags := validateStringInSlice(ingestRouteTypes, false)(parts[0], nil); diags.HasError() {
		return "", "", fmt.Errorf("invalid ingest route type %q", parts[0])
	}
	return parts[0], parts[1], nil
}

func ingestRouteStateID(routeType, id string) string { return routeType + "/" + id }

func ingestRouteIsUnmanageable(route *rest.IngestRouteResource) bool {
	return route.Pipeline == "" || route.ManagedBy != nil
}
