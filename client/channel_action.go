package client

import (
	"github.com/observeinc/terraform-provider-observe/client/internal/meta"
)

type ChannelAction struct {
	ID          string               `json:"id"`
	WorkspaceID string               `json:"workspace"`
	Config      *ChannelActionConfig `json:"config"`
}

type ChannelActionConfig struct {
	Name        string  `json:"name"`
	IconURL     *string `json:"iconUrl"`
	Description *string `json:"description"`

	Webhook *WebhookChannelActionConfig `json:"webhook,omitempty"`
	Email   *EmailChannelActionConfig   `json:"email,omitempty"`
}

type WebhookChannelActionConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers,omitempty"`
}

type EmailChannelActionConfig struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	IsHTML  bool     `json:"isHtml"`
}

func (a *ChannelAction) OID() *OID {
	return &OID{
		Type: TypeChannelAction,
		ID:   a.ID,
	}
}

func (config *ChannelActionConfig) toGQL() (*meta.ChannelActionInput, error) {
	channelActionInput := &meta.ChannelActionInput{
		Name:        &config.Name,
		IconURL:     config.IconURL,
		Description: config.Description,
	}

	if config.Webhook != nil {
		var headers []meta.WebhookHeader
		for k, v := range config.Webhook.Headers {
			headers = append(headers, meta.WebhookHeader{
				Header:        k,
				ValueTemplate: v,
			})
		}

		channelActionInput.Webhook = &meta.WebhookActionInput{
			URLTemplate:  &config.Webhook.URL,
			BodyTemplate: &config.Webhook.Body,
			Method:       &config.Webhook.Method,
			Headers:      &headers,
		}
	}

	if config.Email != nil {
		channelActionInput.Email = &meta.EmailActionInput{
			TargetAddresses: config.Email.To,
			SubjectTemplate: &config.Email.Subject,
			BodyTemplate:    &config.Email.Body,
			IsHTML:          &config.Email.IsHTML,
		}
	}

	return channelActionInput, nil
}

func newChannelAction(a *meta.ChannelAction) (*ChannelAction, error) {
	config := &ChannelActionConfig{
		Name:        a.Name,
		IconURL:     a.IconURL,
		Description: a.Description,
	}

	return &ChannelAction{
		ID:          a.ID.String(),
		WorkspaceID: a.WorkspaceId.String(),
		Config:      config,
	}, nil
}
