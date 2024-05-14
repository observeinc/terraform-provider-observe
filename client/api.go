package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/observeinc/terraform-provider-observe/client/meta"
)

var (
	ErrNotFound = errors.New("not found")

	flagObs2110 = "obs2110" // when set, allow concurrent API calls for foreign keys

	// default backoff values for waiting on app async apply
	syncRetryDuration = time.Second
	syncRetryFactor   = 2.0
	syncRetryCap      = 5 * time.Second
)

// GetDataset returns dataset by ID
func (c *Client) GetDataset(ctx context.Context, id string) (*meta.Dataset, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	return c.Meta.GetDataset(ctx, id)
}

func (c *Client) SaveDataset(ctx context.Context, wsid string, input *meta.DatasetInput, queryInput *meta.MultiStageQueryInput) (*meta.Dataset, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	if c.Config.Source != nil {
		input.Source = c.Config.Source
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}

	return c.Meta.SaveDataset(ctx, wsid, input, queryInput)
}

// DeleteDataset by ID
func (c *Client) DeleteDataset(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteDataset(ctx, id)
}

// GetDataset returns the source dataset by ID
func (c *Client) GetSourceDataset(ctx context.Context, id string) (*meta.Dataset, error) {
	return c.Meta.GetDataset(ctx, id)
}

// CreateSourceDataset creates a new source dataset
func (c *Client) CreateSourceDataset(ctx context.Context, workspaceId string, dataset *meta.DatasetDefinitionInput, table *meta.SourceTableDefinitionInput) (*meta.Dataset, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	return c.Meta.SaveSourceDataset(ctx, workspaceId, dataset, table)
}

// UpdateSourceDataset updates the existing source dataset
func (c *Client) UpdateSourceDataset(ctx context.Context, workspaceId string, id string, dataset *meta.DatasetDefinitionInput, table *meta.SourceTableDefinitionInput) (*meta.Dataset, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	dataset.Dataset.Id = &id
	return c.Meta.SaveSourceDataset(ctx, workspaceId, dataset, table)
}

// GetWorkspace by ID
func (c *Client) GetWorkspace(ctx context.Context, id string) (*meta.Workspace, error) {
	return c.Meta.GetWorkspace(ctx, id)
}

// LookupWorkspace by name.
func (c *Client) LookupWorkspace(ctx context.Context, name string) (*meta.Workspace, error) {
	return c.Meta.LookupWorkspace(ctx, name)
}

// ListWorkspaces.
func (c *Client) ListWorkspaces(ctx context.Context) (workspaces []*meta.Workspace, err error) {
	return c.Meta.ListWorkspaces(ctx)
}

// LookupDataset by name.
func (c *Client) LookupDataset(ctx context.Context, workspaceID string, name string) (*meta.Dataset, error) {
	return c.Meta.LookupDataset(ctx, workspaceID, name)
}

// CreateForeignKey
func (c *Client) CreateForeignKey(ctx context.Context, workspaceID string, input *meta.DeferredForeignKeyInput) (*meta.DeferredForeignKey, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	result, err := c.Meta.CreateDeferredForeignKey(ctx, workspaceID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	if result.Status.ErrorText != "" {
		return result, errors.New(result.Status.ErrorText)
	}

	return result, nil
}

// UpdateForeignKey by ID
func (c *Client) UpdateForeignKey(ctx context.Context, id string, input *meta.DeferredForeignKeyInput) (*meta.DeferredForeignKey, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	result, err := c.Meta.UpdateDeferredForeignKey(ctx, id, input)
	if err != nil {
		return nil, fmt.Errorf("failed to update link: %w", err)
	}

	if result.Status.ErrorText != "" {
		return result, errors.New(result.Status.ErrorText)
	}
	return result, nil
}

// GetForeignKey returns deferred foreign key
func (c *Client) GetForeignKey(ctx context.Context, id string) (*meta.DeferredForeignKey, error) {
	result, err := c.Meta.GetDeferredForeignKey(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get link: %w", err)
	}

	if result.Status.ErrorText != "" {
		return result, fmt.Errorf("link has an error status: %s", result.Status.ErrorText)
	}

	return result, nil
}

// LookupForeignKey by source, target and fields
func (c *Client) LookupForeignKey(ctx context.Context, source string, target string, srcFields []string, dstFields []string) (*meta.DatasetForeignKeysForeignKey, error) {
	dataset, err := c.GetDataset(ctx, source)
	if err != nil {
		return nil, err
	}

	for _, fk := range dataset.ForeignKeys {
		switch {
		case fk.TargetDataset == nil || fk.TargetDataset.String() != target:
			continue
		case !reflect.DeepEqual(fk.SrcFields, srcFields):
			continue
		case !reflect.DeepEqual(fk.DstFields, dstFields):
			continue
		default:
			return &fk, nil
		}
	}

	return nil, ErrNotFound
}

// DeleteForeignKey
func (c *Client) DeleteForeignKey(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteDeferredForeignKey(ctx, id)
}

// GetBookmarkGroup returns bookmarkGroup by ID
func (c *Client) GetBookmarkGroup(ctx context.Context, id string) (*meta.BookmarkGroup, error) {
	return c.Meta.GetBookmarkGroup(ctx, id)
}

// CreateBookmarkGroup creates a bookmark group
func (c *Client) CreateBookmarkGroup(ctx context.Context, workspaceId string, input *meta.BookmarkGroupInput) (*meta.BookmarkGroup, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	input.WorkspaceId = &workspaceId
	return c.Meta.CreateOrUpdateBookmarkGroup(ctx, nil, input)
}

// UpdateBookmarkGroup updates a bookmark group
func (c *Client) UpdateBookmarkGroup(ctx context.Context, id string, input *meta.BookmarkGroupInput) (*meta.BookmarkGroup, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	return c.Meta.CreateOrUpdateBookmarkGroup(ctx, &id, input)
}

// DeleteBookmarkGroup
func (c *Client) DeleteBookmarkGroup(ctx context.Context, id string) error {
	return c.Meta.DeleteBookmarkGroup(ctx, id)
}

// GetBookmark returns bookmark by ID
func (c *Client) GetBookmark(ctx context.Context, id string) (*meta.Bookmark, error) {
	return c.Meta.GetBookmark(ctx, id)
}

// CreateBookmark creates a bookmark group
func (c *Client) CreateBookmark(ctx context.Context, input *meta.BookmarkInput) (*meta.Bookmark, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	return c.Meta.CreateOrUpdateBookmark(ctx, nil, input)
}

// UpdateBookmark updates a bookmark
func (c *Client) UpdateBookmark(ctx context.Context, id string, input *meta.BookmarkInput) (*meta.Bookmark, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	return c.Meta.CreateOrUpdateBookmark(ctx, &id, input)
}

// DeleteBookmark
func (c *Client) DeleteBookmark(ctx context.Context, id string) error {
	return c.Meta.DeleteBookmark(ctx, id)
}

// Observe submits observations
func (c *Client) Observe(ctx context.Context, path string, body io.Reader, tags map[string]string, options ...func(*http.Request)) error {
	return c.Collect.Observe(ctx, path, body, tags, options...)
}

// CreateChannelAction creates a channel action
func (c *Client) CreateChannelAction(ctx context.Context, workspaceId string, input *meta.ActionInput, channels []string) (*meta.ChannelAction, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.CreateChannelAction(ctx, workspaceId, input)
	if err != nil {
		return nil, err
	}
	if err = c.Meta.SetChannelsForChannelAction(ctx, (*result).GetId(), channels); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateChannelAction updates a channel action
func (c *Client) UpdateChannelAction(ctx context.Context, id string, input *meta.ActionInput, channels []string) (*meta.ChannelAction, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.UpdateChannelAction(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if err := c.Meta.SetChannelsForChannelAction(ctx, id, channels); err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteChannelAction
func (c *Client) DeleteChannelAction(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteChannelAction(ctx, id)
}

// GetChannelAction returns channelAction by ID
func (c *Client) GetChannelAction(ctx context.Context, id string) (*meta.ChannelAction, error) {
	return c.Meta.GetChannelAction(ctx, id)
}

// CreateChannel creates a channel
func (c *Client) CreateChannel(ctx context.Context, workspaceId string, input *meta.ChannelInput, monitors []string) (*meta.Channel, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.CreateChannel(ctx, workspaceId, input)
	if err != nil {
		return nil, err
	}
	id := (*result).GetId()

	if err := c.Meta.SetMonitorsForChannel(ctx, id, monitors); err != nil {
		defer c.DeleteChannel(ctx, id)
		return nil, err
	}
	return result, nil
}

// UpdateChannel updates a channel
func (c *Client) UpdateChannel(ctx context.Context, id string, input *meta.ChannelInput, monitors []string) (*meta.Channel, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.UpdateChannel(ctx, id, input)
	if err != nil {
		return nil, err
	}

	if err := c.Meta.SetMonitorsForChannel(ctx, id, monitors); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteChannel
func (c *Client) DeleteChannel(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteChannel(ctx, id)
}

// GetChannel returns channel by ID
func (c *Client) GetChannel(ctx context.Context, id string) (*meta.Channel, error) {
	return c.Meta.GetChannel(ctx, id)
}

func (c *Client) CreateLayeredSettingRecord(ctx context.Context, input *meta.LayeredSettingRecordInput) (*meta.LayeredSettingRecord, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}

	return c.Meta.CreateLayeredSettingRecord(ctx, input)
}

func (c *Client) UpdateLayeredSettingRecord(ctx context.Context, input *meta.LayeredSettingRecordInput) (*meta.LayeredSettingRecord, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.UpdateLayeredSettingRecord(ctx, input)
}

func (c *Client) DeleteLayeredSettingRecord(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteLayeredSettingRecord(ctx, id)
}

func (c *Client) GetLayeredSettingRecord(ctx context.Context, id string) (*meta.LayeredSettingRecord, error) {
	return c.Meta.GetLayeredSettingRecord(ctx, id)
}

// Query for result
func (c *Client) Query(ctx context.Context, stages []*meta.StageInput, params *meta.QueryParams) (result []*meta.TaskResult, err error) {
	return c.Meta.DatasetQueryOutput(ctx, stages, params)
}

// CreateMonitorAction creates a monitor action
func (c *Client) CreateMonitorAction(ctx context.Context, input *meta.MonitorActionInput) (*meta.MonitorAction, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.CreateMonitorAction(ctx, input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateMonitorAction updates a monitor action
func (c *Client) UpdateMonitorAction(ctx context.Context, id string, input *meta.MonitorActionInput) (*meta.MonitorAction, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.UpdateMonitorAction(ctx, id, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteMonitorAction deletes a monitor action
func (c *Client) DeleteMonitorAction(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteMonitorAction(ctx, id)
}

// GetMonitorAction returns a monitor action by ID
func (c *Client) GetMonitorAction(ctx context.Context, id string) (*meta.MonitorAction, error) {
	return c.Meta.GetMonitorAction(ctx, id)
}

func (c *Client) LookupMonitorAction(ctx context.Context, workspaceID string, name string) (*meta.MonitorAction, error) {
	return c.Meta.LookupMonitorAction(ctx, workspaceID, name)
}

// CreateMonitor creates a monitor
func (c *Client) CreateMonitor(ctx context.Context, workspaceId string, input *meta.MonitorInput) (*meta.Monitor, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.Source != nil {
		input.Source = c.Config.Source
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	if input.FreshnessGoal != nil && input.UseDefaultFreshness == nil {
		b := false
		input.UseDefaultFreshness = &b
	}

	return c.Meta.CreateMonitor(ctx, workspaceId, input)
}

// UpdateMonitor updates a monitor
func (c *Client) UpdateMonitor(ctx context.Context, id string, input *meta.MonitorInput) (*meta.Monitor, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.Source != nil {
		input.Source = c.Config.Source
	}
	if input.FreshnessGoal != nil && input.UseDefaultFreshness == nil {
		b := false
		input.UseDefaultFreshness = &b
	}

	return c.Meta.UpdateMonitor(ctx, id, input)
}

// DeleteMonitor deletes a monitor
func (c *Client) DeleteMonitor(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteMonitor(ctx, id)
}

// GetMonitor returns monitor by ID
func (c *Client) GetMonitor(ctx context.Context, id string) (*meta.Monitor, error) {
	return c.Meta.GetMonitor(ctx, id)
}

// CreateMonitorActionAttachment creates a monitor action attachment
func (c *Client) CreateMonitorActionAttachment(ctx context.Context, input *meta.MonitorActionAttachmentInput) (*meta.MonitorActionAttachment, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.CreateMonitorActionAttachment(ctx, input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateMonitorActionAttachment updates a monitor action attachment
func (c *Client) UpdateMonitorActionAttachment(ctx context.Context, id string, input *meta.MonitorActionAttachmentInput) (*meta.MonitorActionAttachment, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.UpdateMonitorActionAttachment(ctx, id, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteMonitorActionAttachment deletes a monitor action attachment
func (c *Client) DeleteMonitorActionAttachment(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteMonitorActionAttachment(ctx, id)
}

// GetMonitorActionAttachment returns a monitor action attachment by ID
func (c *Client) GetMonitorActionAttachment(ctx context.Context, id string) (*meta.MonitorActionAttachment, error) {
	return c.Meta.GetMonitorActionAttachment(ctx, id)
}

// LookupMonitor returns monitor by name
func (c *Client) LookupMonitor(ctx context.Context, workspaceId string, id string) (*meta.Monitor, error) {
	return c.Meta.LookupMonitor(ctx, workspaceId, id)
}

// CreateBoard creates a board
func (c *Client) CreateBoard(ctx context.Context, dsid string, boardType meta.BoardType, input *meta.BoardInput) (*meta.Board, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.Source != nil {
		input.Source = c.Config.Source
	}

	return c.Meta.CreateBoard(ctx, dsid, boardType, input)
}

// UpdateBoard updates a board
func (c *Client) UpdateBoard(ctx context.Context, id string, input *meta.BoardInput) (*meta.Board, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.Source != nil {
		input.Source = c.Config.Source
	}

	return c.Meta.UpdateBoard(ctx, id, input)
}

// DeleteBoard
func (c *Client) DeleteBoard(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteBoard(ctx, id)
}

// GetBoard returns board by ID
func (c *Client) GetBoard(ctx context.Context, id string) (*meta.Board, error) {
	return c.Meta.GetBoard(ctx, id)
}

// CreatePoller creates a poller
func (c *Client) CreatePoller(ctx context.Context, workspaceId string, input *meta.PollerInput) (*meta.Poller, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	return c.Meta.CreatePoller(ctx, workspaceId, input)
}

// UpdatePoller updates a poller
func (c *Client) UpdatePoller(ctx context.Context, id string, input *meta.PollerInput) (*meta.Poller, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	return c.Meta.UpdatePoller(ctx, id, input)
}

// DeletePoller
func (c *Client) DeletePoller(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeletePoller(ctx, id)
}

// GetPoller returns a poller by ID
func (c *Client) GetPoller(ctx context.Context, id string) (*meta.Poller, error) {
	return c.Meta.GetPoller(ctx, id)
}

// CreateWorkspace creates a workspace
func (c *Client) CreateWorkspace(ctx context.Context, input *meta.WorkspaceInput) (*meta.Workspace, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.CreateWorkspace(ctx, input)
}

// UpdateWorkspace updates a workspace
func (c *Client) UpdateWorkspace(ctx context.Context, id string, input *meta.WorkspaceInput) (*meta.Workspace, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.UpdateWorkspace(ctx, id, input)
}

// DeleteWorkspace
func (c *Client) DeleteWorkspace(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteWorkspace(ctx, id)
}

// CreateDatastream creates a datastream
func (c *Client) CreateDatastream(ctx context.Context, workspaceId string, input *meta.DatastreamInput) (*meta.Datastream, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.CreateDatastream(ctx, workspaceId, input)
}

// GetDatastream by ID
func (c *Client) GetDatastream(ctx context.Context, id string) (*meta.Datastream, error) {
	return c.Meta.GetDatastream(ctx, id)
}

// UpdateDatastream updates a datastream
func (c *Client) UpdateDatastream(ctx context.Context, id string, input *meta.DatastreamInput) (*meta.Datastream, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.UpdateDatastream(ctx, id, input)
}

// DeleteDatastream
func (c *Client) DeleteDatastream(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteDatastream(ctx, id)
}

// LookupDatastream by name.
func (c *Client) LookupDatastream(ctx context.Context, workspaceID string, name string) (*meta.Datastream, error) {
	return c.Meta.LookupDatastream(ctx, workspaceID, name)
}

// CreateDatastreamToken creates a datastream token
func (c *Client) CreateDatastreamToken(ctx context.Context, datastreamId string, input *meta.DatastreamTokenInput) (*meta.DatastreamToken, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.CreateDatastreamToken(ctx, datastreamId, input)
}

// GetDatastreamToken by ID
func (c *Client) GetDatastreamToken(ctx context.Context, id string) (*meta.DatastreamToken, error) {
	return c.Meta.GetDatastreamToken(ctx, id)
}

// UpdateDatastreamToken updates a datastream
func (c *Client) UpdateDatastreamToken(ctx context.Context, id string, input *meta.DatastreamTokenInput) (*meta.DatastreamToken, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.UpdateDatastreamToken(ctx, id, input)
}

// DeleteDatastreamToken
func (c *Client) DeleteDatastreamToken(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteDatastreamToken(ctx, id)
}

// CreateWorksheet creates a worksheet
func (c *Client) CreateWorksheet(ctx context.Context, workspaceId string, input *meta.WorksheetInput) (*meta.Worksheet, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	input.WorkspaceId = workspaceId
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}

	return c.Meta.SaveWorksheet(ctx, input)
}

// GetWorksheet by ID
func (c *Client) GetWorksheet(ctx context.Context, id string) (*meta.Worksheet, error) {
	return c.Meta.GetWorksheet(ctx, id)
}

// UpdateWorksheet updates a worksheet
// XXX: this should not have to take workspaceId, but API forces us to
func (c *Client) UpdateWorksheet(ctx context.Context, id string, workspaceId string, input *meta.WorksheetInput) (*meta.Worksheet, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	input.Id = &id
	input.WorkspaceId = workspaceId

	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}

	return c.Meta.SaveWorksheet(ctx, input)
}

// DeleteWorksheet
func (c *Client) DeleteWorksheet(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteWorksheet(ctx, id)
}

func (c *Client) CreateDashboard(ctx context.Context, workspaceId string, input *meta.DashboardInput) (*meta.Dashboard, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	input.WorkspaceId = &workspaceId

	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}

	return c.Meta.SaveDashboard(ctx, input)
}

func (c *Client) GetDashboard(ctx context.Context, id string) (*meta.Dashboard, error) {
	return c.Meta.GetDashboard(ctx, id)
}

// XXX: this should not have to take workspaceId, but API forces us to
func (c *Client) UpdateDashboard(ctx context.Context, id string, workspaceId string, input *meta.DashboardInput) (*meta.Dashboard, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	input.Id = &id
	input.WorkspaceId = &workspaceId

	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}

	return c.Meta.SaveDashboard(ctx, input)
}

func (c *Client) DeleteDashboard(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteDashboard(ctx, id)
}

func (c *Client) GetDefaultDashboard(ctx context.Context, dsid string) (*string, error) {
	return c.Meta.GetDefaultDashboard(ctx, dsid)
}

func (c *Client) SetDefaultDashboard(ctx context.Context, dsid string, dashid string) error {
	return c.Meta.SetDefaultDashboard(ctx, dsid, dashid)
}

func (c *Client) ClearDefaultDashboard(ctx context.Context, dsid string) error {
	return c.Meta.ClearDefaultDashboard(ctx, dsid)
}

// CreateFolder creates a folder
func (c *Client) CreateFolder(ctx context.Context, workspaceId string, input *meta.FolderInput) (*meta.Folder, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.CreateFolder(ctx, workspaceId, input)
}

// UpdateFolder updates a folder
func (c *Client) UpdateFolder(ctx context.Context, id string, input *meta.FolderInput) (*meta.Folder, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.UpdateFolder(ctx, id, input)
}

// DeleteFolder
func (c *Client) DeleteFolder(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteFolder(ctx, id)
}

// GetFolder by ID
func (c *Client) GetFolder(ctx context.Context, id string) (*meta.Folder, error) {
	return c.Meta.GetFolder(ctx, id)
}

// LookupFolder by name.
func (c *Client) LookupFolder(ctx context.Context, workspaceID string, name string) (*meta.Folder, error) {
	return c.Meta.LookupFolder(ctx, workspaceID, name)
}

// CreateApp creates an app
func (c *Client) CreateApp(ctx context.Context, workspaceId string, input *meta.AppInput) (*meta.App, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	result, err := c.Meta.CreateApp(ctx, workspaceId, input)
	if err != nil {
		return nil, err
	}

	// This is tricky. Once we've successfully created the object in API, we
	// want to surface that up to terraform so it can track state.
	// Unfortunately, we still have a lot of API calls to go, all of which can
	// error. We avoid erroring from here on out if possible.
	result, err = c.Meta.UpdateApp(ctx, result.Id, input)
	if err != nil {
		return nil, err
	}

	// we should move this logic to server, so any client requiring synchronous
	// behavior for testing can reuse accordingly.
	duration := syncRetryDuration
	for result.Status.State == "Installing" {
		time.Sleep(duration)
		if r, err := c.Meta.GetApp(ctx, result.Id); err == nil {
			result = r
		} else {
			break
		}

		if nextDuration := duration * time.Duration(syncRetryFactor); nextDuration < syncRetryCap {
			duration = nextDuration
		} else {
			duration = syncRetryCap
		}
	}

	return result, nil
}

// UpdateApp updates a app
func (c *Client) UpdateApp(ctx context.Context, id string, input *meta.AppInput) (*meta.App, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.UpdateApp(ctx, id, input)
	if err != nil {
		return nil, err
	}

	// we should move this logic to server, so any client requiring synchronous
	// behavior for testing can reuse accordingly.
	duration := syncRetryDuration
	for result.Status.State == "Installing" {
		time.Sleep(duration)
		if r, err := c.Meta.GetApp(ctx, result.Id); err == nil {
			result = r
		} else {
			break
		}

		if nextDuration := duration * time.Duration(syncRetryFactor); nextDuration < syncRetryCap {
			duration = nextDuration
		} else {
			duration = syncRetryCap
		}
	}
	return result, nil
}

// DeleteApp
func (c *Client) DeleteApp(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if err := c.Meta.DeleteApp(ctx, id); err != nil {
		return err
	}

	duration := syncRetryDuration
	result, err := c.Meta.GetApp(ctx, id)
	for err == nil && result.Status.State == "Deleting" {
		time.Sleep(duration)
		result, err = c.Meta.GetApp(ctx, id)

		if nextDuration := duration * time.Duration(syncRetryFactor); nextDuration < syncRetryCap {
			duration = nextDuration
		} else {
			duration = syncRetryCap
		}
	}

	if result != nil {
		return fmt.Errorf("failed to delete app: %s", result.Status.State)
	}
	return nil
}

// GetApp by ID
func (c *Client) GetApp(ctx context.Context, id string) (*meta.App, error) {
	return c.Meta.GetApp(ctx, id)
}

// LookupApp by name.
func (c *Client) LookupApp(ctx context.Context, workspaceID string, name string) (*meta.App, error) {
	return c.Meta.LookupApp(ctx, workspaceID, name)
}

func (c *Client) LookupModuleVersions(ctx context.Context, id string) ([]*meta.ModuleVersion, error) {
	return c.Meta.LookupModuleVersions(ctx, id)
}

// CreatePreferredPath creates a preferred path
func (c *Client) CreatePreferredPath(ctx context.Context, workspaceId string, input *meta.PreferredPathInput) (*meta.PreferredPath, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	resultWithStatus, err := c.Meta.CreatePreferredPath(ctx, workspaceId, input)
	if err != nil {
		return nil, err
	}
	if resultWithStatus.Error != nil {
		return nil, errors.New(*resultWithStatus.Error)
	}
	return &resultWithStatus.Path.PreferredPath, nil
}

// UpdatePreferredPath updates a preferred path
func (c *Client) UpdatePreferredPath(ctx context.Context, id string, input *meta.PreferredPathInput) (*meta.PreferredPath, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	resultWithStatus, err := c.Meta.UpdatePreferredPath(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if resultWithStatus.Error != nil {
		return nil, errors.New(*resultWithStatus.Error)
	}
	return &resultWithStatus.Path.PreferredPath, nil
}

// DeletePreferredPath
func (c *Client) DeletePreferredPath(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeletePreferredPath(ctx, id)
}

// GetPreferredPath gets the preferred path by ID. If the path is invalid, an error may be returned in addition to the path object.
func (c *Client) GetPreferredPath(ctx context.Context, id string) (*meta.PreferredPath, error) {
	resultWithStatus, err := c.Meta.GetPreferredPath(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get preferred path: %w", err)
	}

	if resultWithStatus.Error != nil {
		err = errors.New(*resultWithStatus.Error)
	}

	return &resultWithStatus.Path.PreferredPath, err
}

// GetTerraform returns terraform definition
func (c *Client) GetTerraform(ctx context.Context, id string, objType meta.TerraformObjectType) (*meta.TerraformDefinition, error) {
	return c.Meta.GetTerraform(ctx, id, objType)
}

// CreateAppDataSource creates an appdatasource
func (c *Client) CreateAppDataSource(ctx context.Context, input *meta.AppDataSourceInput) (*meta.AppDataSource, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	result, err := c.Meta.CreateAppDataSource(ctx, input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateAppDataSource updates an appdatasource
func (c *Client) UpdateAppDataSource(ctx context.Context, id string, input *meta.AppDataSourceInput) (*meta.AppDataSource, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	result, err := c.Meta.UpdateAppDataSource(ctx, id, input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteAppDataSource
func (c *Client) DeleteAppDataSource(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteAppDataSource(ctx, id)
}

// GetAppDataSource by ID
func (c *Client) GetAppDataSource(ctx context.Context, id string) (*meta.AppDataSource, error) {
	return c.Meta.GetAppDataSource(ctx, id)
}

// CreateRbacGroup creates an rbacgroup
func (c *Client) CreateRbacGroup(ctx context.Context, input *meta.RbacGroupInput) (*meta.RbacGroup, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.CreateRbacGroup(ctx, input)
}

// UpdateRbacGroup updates an rbacgroup
func (c *Client) UpdateRbacGroup(ctx context.Context, id string, input *meta.RbacGroupInput) (*meta.RbacGroup, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.UpdateRbacGroup(ctx, id, input)
}

// DeleteRbacGroup
func (c *Client) DeleteRbacGroup(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteRbacGroup(ctx, id)
}

// GetRbacGroup by ID
func (c *Client) GetRbacGroup(ctx context.Context, id string) (*meta.RbacGroup, error) {
	return c.Meta.GetRbacGroup(ctx, id)
}

// LookupRbacGroup by name.
func (c *Client) LookupRbacGroup(ctx context.Context, name string) (*meta.RbacGroup, error) {
	return c.Meta.LookupRbacGroup(ctx, name)
}

// GetRbacDefaultGroup
func (c *Client) GetRbacDefaultGroup(ctx context.Context) (*meta.RbacGroup, error) {
	return c.Meta.GetRbacDefaultGroup(ctx)
}

// SetRbacDefaultGroup
func (c *Client) SetRbacDefaultGroup(ctx context.Context, id string) error {
	return c.Meta.SetRbacDefaultGroup(ctx, id)
}

// UnsetRbacDefaultGroup
func (c *Client) UnsetRbacDefaultGroup(ctx context.Context) error {
	return c.Meta.UnsetRbacDefaultGroup(ctx)
}

// GetUser by ID
func (c *Client) GetUser(ctx context.Context, id string) (*meta.User, error) {
	return c.Meta.GetUser(ctx, id)
}

// LookupUser by email.
func (c *Client) LookupUser(ctx context.Context, email string) (*meta.User, error) {
	return c.Meta.LookupUser(ctx, email)
}

// CreateRbacGroupmember creates an rbacgroupmember
func (c *Client) CreateRbacGroupmember(ctx context.Context, input *meta.RbacGroupmemberInput) (*meta.RbacGroupmember, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.CreateRbacGroupmember(ctx, input)
}

// UpdateRbacGroupmember updates an rbacgroupmember
func (c *Client) UpdateRbacGroupmember(ctx context.Context, id string, input *meta.RbacGroupmemberInput) (*meta.RbacGroupmember, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.UpdateRbacGroupmember(ctx, id, input)
}

// DeleteRbacGroupmember
func (c *Client) DeleteRbacGroupmember(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteRbacGroupmember(ctx, id)
}

// GetRbacGroupmember by ID
func (c *Client) GetRbacGroupmember(ctx context.Context, id string) (*meta.RbacGroupmember, error) {
	return c.Meta.GetRbacGroupmember(ctx, id)
}

// CreateRbacStatement creates an rbacstatement
func (c *Client) CreateRbacStatement(ctx context.Context, input *meta.RbacStatementInput) (*meta.RbacStatement, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.CreateRbacStatement(ctx, input)
}

// UpdateRbacStatement updates an rbacstatement
func (c *Client) UpdateRbacStatement(ctx context.Context, id string, input *meta.RbacStatementInput) (*meta.RbacStatement, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.UpdateRbacStatement(ctx, id, input)
}

// DeleteRbacStatement
func (c *Client) DeleteRbacStatement(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteRbacStatement(ctx, id)
}

// GetRbacStatement by ID
func (c *Client) GetRbacStatement(ctx context.Context, id string) (*meta.RbacStatement, error) {
	return c.Meta.GetRbacStatement(ctx, id)
}

// CreateFiledrop creates a filedrop
func (c *Client) CreateFiledrop(ctx context.Context, workspaceId string, datastreamId string, input *meta.FiledropInput) (*meta.Filedrop, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	result, err := c.Meta.CreateFiledrop(ctx, workspaceId, datastreamId, input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateFiledrop updates a filedrop
func (c *Client) UpdateFiledrop(ctx context.Context, id string, input *meta.FiledropInput) (*meta.Filedrop, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	if c.Config.ManagingObjectID != nil {
		input.ManagedById = c.Config.ManagingObjectID
	}
	result, err := c.Meta.UpdateFiledrop(ctx, id, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteFiledrop deletes a filedrop
func (c *Client) DeleteFiledrop(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteFiledrop(ctx, id)
}

// GetFiledrop returns a filedrop by ID
func (c *Client) GetFiledrop(ctx context.Context, id string) (*meta.Filedrop, error) {
	return c.Meta.GetFiledrop(ctx, id)
}

func (c *Client) GetSnowflakeOutboundShare(ctx context.Context, id string) (*meta.SnowflakeOutboundShare, error) {
	return c.Meta.GetSnowflakeOutboundShare(ctx, id)
}

func (c *Client) CreateSnowflakeOutboundShare(ctx context.Context, workspaceId string, input *meta.SnowflakeOutboundShareInput) (*meta.SnowflakeOutboundShare, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	return c.Meta.CreateSnowflakeOutboundShare(ctx, workspaceId, input)
}

func (c *Client) UpdateSnowflakeOutboundShare(ctx context.Context, id string, input *meta.SnowflakeOutboundShareInput) (*meta.SnowflakeOutboundShare, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	return c.Meta.UpdateSnowflakeOutboundShare(ctx, id, input)
}

func (c *Client) DeleteSnowflakeOutboundShare(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	return c.Meta.DeleteSnowflakeOutboundShare(ctx, id)
}

func (c *Client) GetDatasetOutboundShare(ctx context.Context, id string) (*meta.DatasetOutboundShare, error) {
	return c.Meta.GetDatasetOutboundShare(ctx, id)
}

func (c *Client) CreateDatasetOutboundShare(ctx context.Context, workspaceId string, datasetId string, shareId string, input *meta.DatasetOutboundShareInput) (*meta.DatasetOutboundShare, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	return c.Meta.CreateDatasetOutboundShare(ctx, workspaceId, datasetId, shareId, input)
}

func (c *Client) UpdateDatasetOutboundShare(ctx context.Context, id string, input *meta.DatasetOutboundShareInput) (*meta.DatasetOutboundShare, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	return c.Meta.UpdateDatasetOutboundShare(ctx, id, input)
}

func (c *Client) DeleteDatasetOutboundShare(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}

	return c.Meta.DeleteDatasetOutboundShare(ctx, id)
}

func (c *Client) CreateCorrelationTag(ctx context.Context, dataset, tag string, path meta.LinkFieldInput) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	err := c.Meta.CreateCorrelationTag(ctx, dataset, tag, path)
	if err != nil {
		return fmt.Errorf("Failed to create correlation tag: %w", err)
	}
	return nil
}

func (c *Client) IsCorrelationTagPresent(ctx context.Context, dataset, tag string, path meta.LinkFieldInput) (bool, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.IsCorrelationTagPresent(ctx, dataset, tag, path)
}

func (c *Client) DeleteCorrelationTag(ctx context.Context, dataset, tag string, path meta.LinkFieldInput) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteCorrelationTag(ctx, dataset, tag, path)
}
