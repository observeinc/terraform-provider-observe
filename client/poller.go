package client

import (
	"encoding/json"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type Poller struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspace"`
	Config      *PollerConfig `json:"config"`
}

type PollerConfig struct {
	Name         string              `json:"name"`
	Retries      *int64              `json:"retries"`
	Interval     *time.Duration      `json:"interval"`
	Tags         map[string]string   `json:"tags,omitempty"`
	Chunk        *PollerChunkConfig  `json:"chunk"`
	PubsubConfig *PollerPubSubConfig `json:"pubsubConfig"`
	HTTPConfig   *PollerHTTPConfig   `json:"httpConfig"`
}

type PollerChunkConfig struct {
	Enabled bool   `json:"enabled"`
	Size    *int64 `json:"size"`
}

type PollerPubSubConfig struct {
	ProjectID      string `json:"projectId"`
	JSONKey        string `json:"jsonKey"`
	SubscriptionID string `json:"subscriptionId"`
}

type PollerHTTPConfig struct {
	Endpoint    string            `json:"endpoint"`
	ContentType string            `json:"contentType"`
	Headers     map[string]string `json:"headers,omitempty"`
}

func (p *Poller) OID() *OID {
	return &OID{
		Type: TypePoller,
		ID:   p.ID,
	}
}

// converts terraform input to GQL friendly input
func (config *PollerConfig) toGQL() (*meta.PollerInput, error) {
	in := &meta.PollerInput{
		Name:    config.Name,
		Retries: config.Retries,
	}
	if config.Interval != nil {
		ts := config.Interval.String()
		in.Interval = &ts
	}
	if config.Chunk != nil {
		in.Chunk = &meta.PollerChunkInput{
			Enabled: config.Chunk.Enabled,
			Size:    config.Chunk.Size,
		}
	}

	tags, err := serializeStringMap(config.Tags)
	if err != nil {
		return nil, err
	}
	in.Tags = tags

	// pubsub
	if config.PubsubConfig != nil {
		in.PubsubConfig = &meta.PollerPubSubInput{
			ProjectID:      config.PubsubConfig.ProjectID,
			JSONKey:        config.PubsubConfig.JSONKey,
			SubscriptionID: config.PubsubConfig.SubscriptionID,
		}
	}

	// http
	if config.HTTPConfig != nil {
		hin := &meta.PollerHTTPInput{
			Endpoint:    config.HTTPConfig.Endpoint,
			ContentType: config.HTTPConfig.ContentType,
		}
		headers, err := serializeStringMap(config.HTTPConfig.Headers)
		if err != nil {
			return nil, err
		}
		hin.Headers = headers
		in.HTTPConfig = hin
	}
	return in, nil
}

// converts an input to a marshaled json string
func serializeStringMap(in interface{}) (*string, error) {
	if in == nil {
		return nil, nil
	}
	if b, err := json.Marshal(in); err != nil {
		return nil, err
	} else {
		m := string(b)
		return &m, nil
	}
}

func makeStringMap(in map[string]interface{}) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, val := range in {
		out[key] = val.(string)
	}
	return out
}

func newPoller(p *meta.Poller) (*Poller, error) {
	var chunkConf *PollerChunkConfig
	if p.Config.Chunk != nil {
		chunkConf = &PollerChunkConfig{
			Enabled: p.Config.Chunk.Enabled,
			Size:    p.Config.Chunk.Size,
		}
	}

	var pubsubConf *PollerPubSubConfig
	if p.Config.PubSubConfig != nil {
		key, err := serializeStringMap(p.Config.PubSubConfig.JSONKey)
		if err != nil {
			return nil, err
		}
		pubsubConf = &PollerPubSubConfig{
			ProjectID:      p.Config.PubSubConfig.ProjectID,
			SubscriptionID: p.Config.PubSubConfig.SubscriptionID,
			JSONKey:        *key,
		}
	}

	var httpConf *PollerHTTPConfig
	if p.Config.HTTPConfig != nil {
		httpConf = &PollerHTTPConfig{
			Endpoint:    p.Config.HTTPConfig.Endpoint,
			ContentType: p.Config.HTTPConfig.ContentType,
			Headers:     makeStringMap(p.Config.HTTPConfig.Headers),
		}
	}

	pc := &PollerConfig{
		Name:         p.Config.Name,
		Retries:      p.Config.Retries,
		Interval:     p.Config.Interval,
		Tags:         makeStringMap(p.Config.Tags),
		Chunk:        chunkConf,
		PubsubConfig: pubsubConf,
		HTTPConfig:   httpConf,
	}
	//TODO: include workspaceId?
	out := &Poller{
		ID:     p.ID.String(),
		Config: pc,
	}
	return out, nil
}
