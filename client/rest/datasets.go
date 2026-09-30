package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// datasetLabelBatchSize is the server's maximum /v1/datasets page size.
const datasetLabelBatchSize = 100

type datasetLabelListResponse struct {
	Datasets *[]struct {
		Id    *string `json:"id"`
		Label *string `json:"label"`
	} `json:"datasets"`
	Meta *struct {
		TotalCount *int64 `json:"totalCount"`
	} `json:"meta"`
}

// LookupDatasetLabels returns id→label for the datasets among ids that exist
// and are visible to the caller. IDs must be positive; duplicates are allowed.
// Each request filters on at most 100 IDs; an empty input makes no request.
func (c *Client) LookupDatasetLabels(ctx context.Context, ids []int64) (map[int64]string, error) {
	unique := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("invalid dataset id %d", id)
		}
		unique[id] = struct{}{}
	}
	sorted := make([]int64, 0, len(unique))
	for id := range unique {
		sorted = append(sorted, id)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	labels := make(map[int64]string, len(sorted))
	for start := 0; start < len(sorted); start += datasetLabelBatchSize {
		end := min(start+datasetLabelBatchSize, len(sorted))
		if err := c.lookupDatasetLabelBatch(ctx, sorted[start:end], labels); err != nil {
			return nil, err
		}
	}
	return labels, nil
}

func (c *Client) lookupDatasetLabelBatch(ctx context.Context, batch []int64, labels map[int64]string) error {
	requested := make(map[int64]bool, len(batch))
	literals := make([]string, len(batch))
	for i, id := range batch {
		requested[id] = true
		literals[i] = strconv.FormatInt(id, 10)
	}
	filter := "id in [" + strings.Join(literals, ", ") + "]"

	seen := make(map[int64]bool, len(batch))
	for offset := 0; ; {
		page, total, err := c.listDatasetLabelPage(ctx, filter, offset)
		if err != nil {
			return err
		}
		if total < 0 || total > int64(len(batch)) {
			return fmt.Errorf("dataset lookup: unexpected totalCount %d for %d ids", total, len(batch))
		}
		for id, label := range page {
			if !requested[id] {
				return fmt.Errorf("dataset lookup: response contains unrequested id %d", id)
			}
			if seen[id] {
				return fmt.Errorf("dataset lookup: response repeats id %d", id)
			}
			seen[id] = true
			labels[id] = label
		}
		offset += len(page)
		if int64(offset) >= total {
			return nil
		}
		if len(page) == 0 {
			return fmt.Errorf("dataset lookup: empty page at offset %d of %d", offset, total)
		}
	}
}

func (c *Client) listDatasetLabelPage(ctx context.Context, filter string, offset int) (map[int64]string, int64, error) {
	query := url.Values{
		"filter":  {filter},
		"limit":   {strconv.Itoa(datasetLabelBatchSize)},
		"offset":  {strconv.Itoa(offset)},
		"orderBy": {"id"},
	}
	resp, err := c.requestWithContext(ctx, http.MethodGet, "/v1/datasets?"+query.Encode(), "", nil)
	if err != nil {
		return nil, 0, fmt.Errorf("dataset lookup: %w", err)
	}
	defer resp.Body.Close()

	var body datasetLabelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, 0, fmt.Errorf("dataset lookup: decoding response: %w", err)
	}
	if body.Datasets == nil || body.Meta == nil || body.Meta.TotalCount == nil {
		return nil, 0, fmt.Errorf("dataset lookup: response missing datasets or meta.totalCount")
	}
	page := make(map[int64]string, len(*body.Datasets))
	for _, ds := range *body.Datasets {
		if ds.Id == nil || ds.Label == nil {
			return nil, 0, fmt.Errorf("dataset lookup: dataset missing id or label")
		}
		id, err := strconv.ParseInt(*ds.Id, 10, 64)
		if err != nil || strconv.FormatInt(id, 10) != *ds.Id {
			return nil, 0, fmt.Errorf("dataset lookup: invalid dataset id %q", *ds.Id)
		}
		if _, dup := page[id]; dup {
			return nil, 0, fmt.Errorf("dataset lookup: response repeats id %d", id)
		}
		page[id] = *ds.Label
	}
	return page, *body.Meta.TotalCount, nil
}
