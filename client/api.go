package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

var (
	ErrNotFound = errors.New("not found")

	flagObs2110 = "obs2110" // when set, allow concurrent API calls for foreign keys
)

// GetDataset returns dataset by ID
func (c *Client) GetDataset(ctx context.Context, id string) (*Dataset, error) {
	result, err := c.Meta.GetDataset(ctx, id)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
}

// CreateDataset creates dataset
func (c *Client) CreateDataset(ctx context.Context, workspaceId string, config *DatasetConfig) (*Dataset, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	datasetInput, transformInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	if c.Config.Source != nil {
		datasetInput.Source = c.Config.Source
	}

	result, err := c.Meta.SaveDataset(ctx, workspaceId, datasetInput, transformInput)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
}

// UpdateDataset updates existing dataset
func (c *Client) UpdateDataset(ctx context.Context, workspaceId string, id string, config *DatasetConfig) (*Dataset, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	datasetInput, transformInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	if c.Config.Source != nil {
		datasetInput.Source = c.Config.Source
	}

	datasetInput.ID = toObjectPointer(&id)
	result, err := c.Meta.SaveDataset(ctx, workspaceId, datasetInput, transformInput)
	if err != nil {
		return nil, err
	}
	return newDataset(result)
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
func (c *Client) GetSourceDataset(ctx context.Context, id string) (*SourceDataset, error) {
	result, err := c.Meta.GetDataset(ctx, id)
	if err != nil {
		return nil, err
	}
	return newSourceDataset(result)
}

// CreateSourceDataset creates a new source dataset
func (c *Client) CreateSourceDataset(ctx context.Context, workspaceId string, config *SourceDatasetConfig) (*SourceDataset, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	dataset, table := config.toGQL()
	result, err := c.Meta.SaveSourceDataset(ctx, workspaceId, dataset, table)
	if err != nil {
		return nil, err
	}
	return newSourceDataset(result)
}

// UpdateSourceDataset updates the existing source dataset
func (c *Client) UpdateSourceDataset(ctx context.Context, workspaceId string, id string, config *SourceDatasetConfig) (*SourceDataset, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	dataset, table := config.toGQL()
	dataset.Dataset.ID = toObjectPointer(&id)
	result, err := c.Meta.SaveSourceDataset(ctx, workspaceId, dataset, table)
	if err != nil {
		return nil, err
	}
	return newSourceDataset(result)
}

// GetWorkspace by ID
func (c *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	result, err := c.Meta.GetWorkspace(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}
	return newWorkspace(result)
}

// LookupWorkspace by name.
func (c *Client) LookupWorkspace(ctx context.Context, name string) (*Workspace, error) {
	result, err := c.Meta.LookupWorkspace(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup workspace: %w", err)
	}
	return newWorkspace(result)
}

// ListWorkspaces.
func (c *Client) ListWorkspaces(ctx context.Context) (workspaces []*Workspace, err error) {
	result, err := c.Meta.ListWorkspaces(ctx)
	if err != nil {
		return
	}

	for _, w := range result {
		if ws, err := newWorkspace(w); err != nil {
			return nil, err
		} else {
			workspaces = append(workspaces, ws)
		}
	}

	return
}

// LookupDataset by name.
func (c *Client) LookupDataset(ctx context.Context, workspaceID string, name string) (*Dataset, error) {
	result, err := c.Meta.LookupDataset(ctx, workspaceID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup dataset: %w", err)
	}
	return newDataset(result)
}

// CreateForeignKey
func (c *Client) CreateForeignKey(ctx context.Context, workspaceID string, config *ForeignKeyConfig) (*ForeignKey, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	foreignKeyInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.Meta.CreateDeferredForeignKey(ctx, workspaceID, foreignKeyInput)
	if err != nil {
		return nil, err
	}

	if result.Status.ErrorText != "" {
		// call internal API directly since DeleteForeignKey() acquires lock
		c.Meta.DeleteDeferredForeignKey(ctx, result.ID.String())
		return nil, fmt.Errorf(result.Status.ErrorText)
	}
	return newForeignKey(result)
}

// UpdateForeignKey by ID
func (c *Client) UpdateForeignKey(ctx context.Context, id string, config *ForeignKeyConfig) (*ForeignKey, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	foreignKeyInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.Meta.UpdateDeferredForeignKey(ctx, id, foreignKeyInput)
	if err != nil {
		return nil, err
	}

	if result.Status.ErrorText != "" {
		return nil, fmt.Errorf(result.Status.ErrorText)
	}
	return newForeignKey(result)
}

// GetForeignKey returns deferred foreign key
func (c *Client) GetForeignKey(ctx context.Context, id string) (*ForeignKey, error) {
	result, err := c.Meta.GetDeferredForeignKey(ctx, id)
	if err != nil {
		return nil, err
	}

	return newForeignKey(result)
}

// LookupForeignKey by source, target and fields
func (c *Client) LookupForeignKey(ctx context.Context, source string, target string, srcFields []string, dstFields []string) (*ForeignKey, error) {
	dataset, err := c.GetDataset(ctx, source)
	if err != nil {
		return nil, err
	}

	var matched *ForeignKeyConfig

	for _, fk := range dataset.ForeignKeys {
		switch {
		case fk.Target == nil || *fk.Target != target:
			continue
		case !reflect.DeepEqual(fk.SrcFields, srcFields):
			continue
		case !reflect.DeepEqual(fk.DstFields, dstFields):
			continue
		default:
			matched = &fk
			break
		}
	}

	if matched == nil {
		return nil, ErrNotFound
	}

	matched.Source = &dataset.ID

	return &ForeignKey{
		Workspace: dataset.WorkspaceID,
		Config:    matched,
	}, nil
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
func (c *Client) GetBookmarkGroup(ctx context.Context, id string) (*BookmarkGroup, error) {
	result, err := c.Meta.GetBookmarkGroup(ctx, id)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// CreateBookmarkGroup creates a bookmark group
func (c *Client) CreateBookmarkGroup(ctx context.Context, workspaceId string, config *BookmarkGroupConfig) (*BookmarkGroup, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	bookmarkGroupInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	bookmarkGroupInput.WorkspaceID = toObjectPointer(&workspaceId)
	result, err := c.Meta.CreateOrUpdateBookmarkGroup(ctx, nil, bookmarkGroupInput)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// UpdateBookmarkGroup updates a bookmark group
func (c *Client) UpdateBookmarkGroup(ctx context.Context, id string, config *BookmarkGroupConfig) (*BookmarkGroup, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	bookmarkGroupInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.CreateOrUpdateBookmarkGroup(ctx, &id, bookmarkGroupInput)
	if err != nil {
		return nil, err
	}
	return newBookmarkGroup(result)
}

// DeleteBookmarkGroup
func (c *Client) DeleteBookmarkGroup(ctx context.Context, id string) error {
	return c.Meta.DeleteBookmarkGroup(ctx, id)
}

// GetBookmark returns bookmark by ID
func (c *Client) GetBookmark(ctx context.Context, id string) (*Bookmark, error) {
	result, err := c.Meta.GetBookmark(ctx, id)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
}

// CreateBookmark creates a bookmark group
func (c *Client) CreateBookmark(ctx context.Context, config *BookmarkConfig) (*Bookmark, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	bookmarkInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.CreateOrUpdateBookmark(ctx, nil, bookmarkInput)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
}

// UpdateBookmark updates a bookmark
func (c *Client) UpdateBookmark(ctx context.Context, id string, config *BookmarkConfig) (*Bookmark, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	bookmarkInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.CreateOrUpdateBookmark(ctx, &id, bookmarkInput)
	if err != nil {
		return nil, err
	}
	return newBookmark(result)
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
func (c *Client) CreateChannelAction(ctx context.Context, workspaceId string, config *ChannelActionConfig) (*ChannelAction, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	channelActionInput, channels, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.CreateChannelAction(ctx, workspaceId, channelActionInput)
	if err != nil {
		return nil, err
	}

	if err := c.Meta.SetChannelsForChannelAction(ctx, result.ID.String(), channels); err != nil {
		defer c.DeleteChannelAction(ctx, result.ID.String())
		return nil, err
	}

	return newChannelAction(result)
}

// UpdateChannelAction updates a bookmark
func (c *Client) UpdateChannelAction(ctx context.Context, id string, config *ChannelActionConfig) (*ChannelAction, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	channelActionInput, channels, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.UpdateChannelAction(ctx, id, channelActionInput)
	if err != nil {
		return nil, err
	}

	if err := c.Meta.SetChannelsForChannelAction(ctx, id, channels); err != nil {
		return nil, err
	}

	return newChannelAction(result)
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
func (c *Client) GetChannelAction(ctx context.Context, id string) (*ChannelAction, error) {
	result, err := c.Meta.GetChannelAction(ctx, id)
	if err != nil {
		return nil, err
	}
	return newChannelAction(result)
}

// CreateChannel creates a channel
func (c *Client) CreateChannel(ctx context.Context, workspaceId string, config *ChannelConfig) (*Channel, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	channelInput, monitors, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.CreateChannel(ctx, workspaceId, channelInput)
	if err != nil {
		return nil, err
	}

	if err := c.Meta.SetMonitorsForChannel(ctx, result.ID.String(), monitors); err != nil {
		defer c.DeleteChannel(ctx, result.ID.String())
		return nil, err
	}

	return newChannel(result)
}

// UpdateChannel updates a channel
func (c *Client) UpdateChannel(ctx context.Context, id string, config *ChannelConfig) (*Channel, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	channelInput, monitors, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.UpdateChannel(ctx, id, channelInput)
	if err != nil {
		return nil, err
	}

	if err := c.Meta.SetMonitorsForChannel(ctx, id, monitors); err != nil {
		return nil, err
	}

	return newChannel(result)
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
func (c *Client) GetChannel(ctx context.Context, id string) (*Channel, error) {
	result, err := c.Meta.GetChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	return newChannel(result)
}

// Query for result
func (c *Client) Query(ctx context.Context, config *QueryConfig) (result *QueryResult, err error) {
	stages, params, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	gqlResult, err := c.Meta.DatasetQueryOutput(ctx, stages, params)
	if err != nil {
		return nil, err
	}

	return newQueryResult(gqlResult)
}

// CreateMonitor creates a monitor
func (c *Client) CreateMonitor(ctx context.Context, workspaceId string, config *MonitorConfig) (*Monitor, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	monitorInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.Meta.CreateMonitor(ctx, workspaceId, monitorInput)
	if err != nil {
		return nil, err
	}
	return newMonitor(result)
}

// UpdateMonitor updates a monitor
func (c *Client) UpdateMonitor(ctx context.Context, id string, config *MonitorConfig) (*Monitor, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	monitorInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.Meta.UpdateMonitor(ctx, id, monitorInput)
	if err != nil {
		return nil, err
	}
	return newMonitor(result)
}

// DeleteMonitor
func (c *Client) DeleteMonitor(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteMonitor(ctx, id)
}

// GetMonitor returns monitor by ID
func (c *Client) GetMonitor(ctx context.Context, id string) (*Monitor, error) {
	result, err := c.Meta.GetMonitor(ctx, id)
	if err != nil {
		return nil, err
	}
	return newMonitor(result)
}

// LookupMonitor returns monitor by name
func (c *Client) LookupMonitor(ctx context.Context, workspaceId string, id string) (*Monitor, error) {
	result, err := c.Meta.LookupMonitor(ctx, workspaceId, id)
	if err != nil {
		return nil, err
	}
	return newMonitor(result)
}

// CreateBoard creates a board
func (c *Client) CreateBoard(ctx context.Context, dataset *OID, boardType BoardType, config *BoardConfig) (*Board, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	boardInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.Meta.CreateBoard(ctx, dataset.ID, boardType, boardInput)
	if err != nil {
		return nil, err
	}
	return newBoard(result)
}

// UpdateBoard updates a board
func (c *Client) UpdateBoard(ctx context.Context, id string, config *BoardConfig) (*Board, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	boardInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.Meta.UpdateBoard(ctx, id, boardInput)
	if err != nil {
		return nil, err
	}
	return newBoard(result)
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
func (c *Client) GetBoard(ctx context.Context, id string) (*Board, error) {
	result, err := c.Meta.GetBoard(ctx, id)
	if err != nil {
		return nil, err
	}
	return newBoard(result)
}

// CreatePoller creates a poller
func (c *Client) CreatePoller(ctx context.Context, workspaceId string, config *PollerConfig) (*Poller, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	pollerInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.Meta.CreatePoller(ctx, workspaceId, pollerInput)
	if err != nil {
		return nil, err
	}
	return newPoller(result)
}

// UpdatePoller updates a poller
func (c *Client) UpdatePoller(ctx context.Context, id string, config *PollerConfig) (*Poller, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	pollerInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}
	result, err := c.Meta.UpdatePoller(ctx, id, pollerInput)
	if err != nil {
		return nil, err
	}
	return newPoller(result)
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
func (c *Client) GetPoller(ctx context.Context, id string) (*Poller, error) {
	result, err := c.Meta.GetPoller(ctx, id)
	if err != nil {
		return nil, err
	}
	return newPoller(result)
}

// CreateWorkspace creates a workspace
func (c *Client) CreateWorkspace(ctx context.Context, config *WorkspaceConfig) (*Workspace, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	workspaceInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.CreateWorkspace(ctx, workspaceInput)
	if err != nil {
		return nil, err
	}

	return newWorkspace(result)
}

// UpdateWorkspace updates a workspace
func (c *Client) UpdateWorkspace(ctx context.Context, id string, config *WorkspaceConfig) (*Workspace, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	workspaceInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.UpdateWorkspace(ctx, id, workspaceInput)
	if err != nil {
		return nil, err
	}

	return newWorkspace(result)
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
func (c *Client) CreateDatastream(ctx context.Context, workspaceId string, config *DatastreamConfig) (*Datastream, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	datastreamInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.CreateDatastream(ctx, workspaceId, datastreamInput)
	if err != nil {
		return nil, err
	}

	return newDatastream(result)
}

// GetDatastream by ID
func (c *Client) GetDatastream(ctx context.Context, id string) (*Datastream, error) {
	result, err := c.Meta.GetDatastream(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get datastream: %w", err)
	}
	return newDatastream(result)
}

// UpdateDatastream updates a datastream
func (c *Client) UpdateDatastream(ctx context.Context, id string, config *DatastreamConfig) (*Datastream, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	datastreamInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.UpdateDatastream(ctx, id, datastreamInput)
	if err != nil {
		return nil, err
	}

	return newDatastream(result)
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
func (c *Client) LookupDatastream(ctx context.Context, workspaceID string, name string) (*Datastream, error) {
	result, err := c.Meta.LookupDatastream(ctx, workspaceID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup datastream: %w", err)
	}
	return newDatastream(result)
}

// CreateDatastreamToken creates a datastream token
func (c *Client) CreateDatastreamToken(ctx context.Context, datastreamId string, config *DatastreamTokenConfig) (*DatastreamToken, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	datastreamTokenInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.CreateDatastreamToken(ctx, datastreamId, datastreamTokenInput)
	if err != nil {
		return nil, err
	}

	return newDatastreamToken(result)
}

// GetDatastreamToken by ID
func (c *Client) GetDatastreamToken(ctx context.Context, id string) (*DatastreamToken, error) {
	result, err := c.Meta.GetDatastreamToken(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get datastream token: %w", err)
	}
	return newDatastreamToken(result)
}

// UpdateDatastreamToken updates a datastream
func (c *Client) UpdateDatastreamToken(ctx context.Context, id string, config *DatastreamTokenConfig) (*DatastreamToken, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	datastreamTokenInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	result, err := c.Meta.UpdateDatastreamToken(ctx, id, datastreamTokenInput)
	if err != nil {
		return nil, err
	}

	return newDatastreamToken(result)
}

// DeleteDatastreamToken
func (c *Client) DeleteDatastreamToken(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteDatastreamToken(ctx, id)
}

// CreateWorksheet creates a datastream token
func (c *Client) CreateWorksheet(ctx context.Context, workspaceId string, config *WorksheetConfig) (*Worksheet, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	worksheetInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	worksheetInput.SetWorkspaceID(workspaceId)

	result, err := c.Meta.SaveWorksheet(ctx, worksheetInput)
	if err != nil {
		return nil, err
	}

	return newWorksheet(result)
}

// GetWorksheet by ID
func (c *Client) GetWorksheet(ctx context.Context, id string) (*Worksheet, error) {
	result, err := c.Meta.GetWorksheet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get worksheet: %w", err)
	}
	return newWorksheet(result)
}

// UpdateWorksheet updates a worksheet
// XXX: this should not have to take workspaceId, but API forces us to
func (c *Client) UpdateWorksheet(ctx context.Context, id string, workspaceId string, config *WorksheetConfig) (*Worksheet, error) {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	worksheetInput, err := config.toGQL()
	if err != nil {
		return nil, err
	}

	worksheetInput.SetID(id)
	worksheetInput.SetWorkspaceID(workspaceId)

	result, err := c.Meta.SaveWorksheet(ctx, worksheetInput)
	if err != nil {
		return nil, err
	}

	return newWorksheet(result)
}

// DeleteWorksheet
func (c *Client) DeleteWorksheet(ctx context.Context, id string) error {
	if !c.Flags[flagObs2110] {
		c.obs2110.Lock()
		defer c.obs2110.Unlock()
	}
	return c.Meta.DeleteWorksheet(ctx, id)
}
