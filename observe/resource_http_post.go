package observe

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	observe "github.com/observeinc/terraform-provider-observe/client"
)

func resourceHTTPPost() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHTTPPostCreate,
		ReadContext:   resourceNoop,
		UpdateContext: resourceNoop,
		DeleteContext: resourceNoop,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			var (
				acked      = d.Get("acked").(string)
				refresh, _ = d.Get("refresh").(string)
			)

			if acked == "" || refresh == "" {
				return nil
			}

			ackedTime, _ := time.Parse(time.RFC3339, acked)
			refreshDuration, _ := time.ParseDuration(refresh)

			if time.Now().After(ackedTime.Add(refreshDuration)) {
				return d.SetNewComputed("acked")
			}
			return nil
		},
		Schema: map[string]*schema.Schema{
			"path": {
				Type:             schema.TypeString,
				Default:          "/terraform/data",
				ValidateDiagFunc: validatePath,
				Optional:         true,
				ForceNew:         true,
				Description:      "Path under which to submit observations",
			},
			"data": {
				Type:             schema.TypeString,
				ValidateDiagFunc: validateStringIsJSON,
				Required:         true,
				ForceNew:         true,
				Description:      "Data to submit to Observe collector",
			},
			"tags": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ForceNew:    true,
				Optional:    true,
				Description: "Tags to set on submitted observations",
			},
			"id_tag": {
				Type:        schema.TypeString,
				Default:     "tf_id",
				Optional:    true,
				ForceNew:    true,
				Description: "Key used to tag submitted observations with unique ID. Set to empty string to omit tag",
			},
			"content_type": {
				Type: schema.TypeString,
				// TODO: only support JSON for now, we'll need to validate data based on content-type
				ValidateDiagFunc: validateStringInSlice([]string{
					"application/json",
				}, false),
				Default:     "application/json",
				Optional:    true,
				ForceNew:    true,
				Description: "Content Type for HTTP POST request",
			},
			"headers": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				ForceNew:    true,
				Description: "Additional HTTP headers",
			},
			// refresh specifies the duration before which we should consider
			// this resource to be stale, at which point we should "recreate" it.
			//
			// Coupling an observation "resource" with a refresh rate allows
			// for usecases such as resubmitting static data periodically in
			// order to create a known resource. Currently users are forced to
			// submit data on a cronjob with some outside script.
			"refresh": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validateTimeDuration,
				Description:      "After refresh duration has elapsed, HTTP POST will be resubmitted",
			},
			// acked stores the timestamp of the submission time for our
			// observations.
			"acked": {
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "Timestamp of submission",
			},
		},
	}
}

func resourceHTTPPostCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	client := meta.(*observe.Client)

	var (
		path        = data.Get("path").(string)
		rawData     = strings.NewReader(data.Get("data").(string))
		rawTags     = data.Get("tags").(map[string]interface{})
		rawHeaders  = data.Get("headers").(map[string]interface{})
		idTag       = data.Get("id_tag").(string)
		contentType = data.Get("content_type").(string)
	)

	tags := make(map[string]string, len(rawTags))
	for k, v := range rawTags {
		tags[k] = v.(string)
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return diag.Errorf("failed to generate id: %s", err)
	}

	id := fmt.Sprintf("%x", string(b))

	if idTag != "" {
		tags[idTag] = id
	}

	var requestOptions []func(*http.Request)

	requestOptions = append(requestOptions, func(req *http.Request) {
		req.Header.Set("Content-Type", contentType)

		for k, v := range rawHeaders {
			// Note: we allow users to override content-type in custom headers.
			// While not recommended, this opens an escape hatch for testing
			// new content-types, or testing the API response to broken inputs.
			req.Header.Set(k, v.(string))
		}
	})

	if err := client.Observe(ctx, path, rawData, tags, requestOptions...); err != nil {
		return diag.Errorf("failed to submit observations: %s", err)
	}

	data.Set("acked", time.Now().UTC().Format(time.RFC3339))
	data.SetId(id)
	return nil
}

func resourceHTTPPostUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	return nil
}

func resourceNoop(ctx context.Context, data *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	return diags
}
