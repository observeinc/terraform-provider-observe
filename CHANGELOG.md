Current releases and their changelogs are available via GitHub [Releases](https://github.com/observeinc/terraform-provider-observe/releases). Past releases are listed below for reference. 

_This document is frozen and will receive no further updates._

<hr>

## 0.13.12 (2023-06-01)

* dataset: make error matcher whitespace-agnostic
* dataset: fix overly specific error matcher in test
* [OB-16320] Allow the provider to set acceleration_disabled on datasets
* feat: support auth_scheme in observe_poller
* feat: add monitor actions resource
* [OB-17369] Stop using sourceTable.tableName and instead use sourceTable.partitions.name

## 0.13.11 (2023-05-05)
* fix: fix tests
* fix: always set optional owner property in rbac_statement
* [OB-14910] Dataset name instead of label
* dataset (source): validate that id and name are not empty strings
* layered_setting_record: add datastream as supported target type
* [OB-16012] datastream / dataset: add validation rules to the name
* ci: pass in pattern to sweeper
* fix: fix sweeper test for datasets

## 0.13.10 (2023-04-19)
* rbac_default_group: add resource.observe_rbac_default_group
* docs(rbac): add examples
* rbac_statement: add resource.observe_rbac_statement
* monitor: add missing fields in data source definition
* fix: adjust SaveMode in dataset
* rbac_group_member: add resource.observe_rbac_group_member
* rbac_group: add resource resource.observe_rbac_group
* user: add datasource data.observe_user
* rbac_group: add datasource data.observe_rbac_group
* workspace: avoid fetching datasets when getting workspace
* docs(poller): add example (HTTP)

## 0.13.9 (2023-03-02)

* poller: Allow a request body to be used HTTP requests
* docs: add example for default_dashboard
* docs: add examples for bookmarks/groups
* docs: add example for observe_folder

## 0.13.8 (2023-02-27)

* fix: restore marshaling of floats as strings

## 0.13.7 (2023-02-24)

* dashboard: marshal float64 as float64 instead of string
* dashboard: handle defaultValue.int64 as int string

## 0.13.6 (2023-02-24)

* dashboard: fix defaultValue.primaryKeyValue[].name
* bookmarks: add managedById

## 0.13.5 (2023-02-23)

* build: fix `docker-package` target
* monitor: add support for setting reminders
* bookmark_group: add description field
* build: fix `pwd` in `check-generated`
* build: fix errors during `go generate` in Jenkins
* dashboard: implement custom Go type for handling parameter defaultValue
* bookmark_group: reference presentation settings from GQL schema
* gql: update schema

## 0.13.4 (2023-02-16)

* docs: fix import examples
* docs(app): add examples
* docs(datastream): add examples for datastreams and tokens
* app: replace hardcoded state name with enum consts
* provider: update descriptions for config attributes
* app: set `name` attribute
* test: create JUnit XML report for acceptance tests
* bookmark: mention dashboard support in description
* poller: add `disabled` attribute
* poller: deprecate old HTTP attributes
* monitor: Implement `is_template` field for data/import

## 0.13.3 (2023-02-02)

* bookmark: support dashboards
* dashboard: fix corrupted parameters defaultValue when reading from API

## 0.13.2 (2023-01-30)

* fix: Terraform CLI version in user-agent header
* docs: add missing descriptions to resources
* make: fix variable name for docs repo
* monitor: implement `comment` field for data/import
* fix: warn about observe_link status error
* poller: implement `kind` attribute
* make: remove generate before build
* test: remove custom terraform installer
* docs: remove outdated example
* client: handle GraphQL generation via go generate
* make: remove broken website targets
* sweep: remove datastreams before datasets
* docs: go generate, fail on diff
* make: build static binary without cgo
* feat: Added support for ManagedById to PreferredPath
* fix: surface error on update app
* test/sweep: add worksheet sweeper

## 0.13.1 (2022-12-16)

* test/sweep: add bookmark sweeper
* make: add docker-sweep target
* rm observe/.client.go.swp
* test/sweepers: simplify names
* Remove dependency on built-in observation dataset
* test/sweep: add sweeper for preferred_paths
* feat: add comment argument to resource_monitor for long form descriptions
* add s/master-periodic
* test/sweep: delete pollers before datastreams
* Rename layered_setting_record files to match resource name
* tests: serialize app tests
* test: use -json to prevent output buffering
* fix: remove datasets from workspace lookup

## 0.13.0 (2022-11-14)

* fix: preferred_path: allow "error" status on read
* feat: add named link support to observe_preferred_path
* feat: dashboard link support
* feat: add support for model_dfk_master.dfk_managed_by_id
* feat: add is_template for monitors
* fix: use workspaceId
* chore: fix a go vet complaint
* chore: Keep up to date with upstream schema
* feat: Add AppDataSource support

## 0.12.0 (2022-09-12)

* fix: send null request method rather than empty string
* fix: update layered setting support
* fix: separate folder from workspace
* docs: add makefile, regenerate
* fix: update after layered settings API refactor
* fix(dashboard): add description to terraform
* fix: relax primary_key for promote monitor
* docs: wire in docs for link
* docs: wire dataset and monitor descriptions
* docs: break out descriptions
* docs: update provider docs, regenerate.
* ci: fix env var passed through for test
* feat: rename OBSERVE_TOKEN to OBSERVE_API_TOKEN
* feat: remove `proxy` support in client
* fix: update go dependencies

## 0.11.3 (2022-09-12)

* feat: add support for 'OBSERVE_SOURCE_FORMAT' env variable
* feat: add observe_terraform data source
* fix: adjust channel action default rate limit
* fix: adjust channel action rate limit tests
* fix: set managedBy on poller updates

## 0.11.2 (2022-09-01)

* feat: set managedById on pollers
* feat: add schema for app data sources and managedById for pollers

## 0.11.1 (2022-08-31)

* fix: set workspace in data sources
* fix: require folder with app name in data source

## 0.11.0 (2022-08-23)

* feat: allow workspace to be specified instead of folder when creating preferred paths
* fix: make workspace id optional
* feat: support config overrides

## 0.10.5 (2022-08-16)
* fix: set overwriteSource on monitor changes
* fix: surface internalError on app install / update

## 0.10.4 (2022-08-10)
* fix: include version in dataset OID

## 0.10.3 (2022-08-10)
* fix: elide path_cost if absent
* fix: set overwriteSource on dataset changes

## 0.10.2 (2022-08-09)
* fix: add back support for email and webhook channel actions

## 0.10.1 (2022-08-08)
* fix: handle input clobbering

## 0.10.0 (2022-08-08)
* chore: switch to auto-generated GraphQL client

## 0.9.2 (2022-08-03)
* fix(monitor): facet_values should never be null

## 0.9.1 (2022-08-02)
* fix(monitor): is_null, is_not_null facet config

## 0.9.0 (2022-08-01)
* feat: add ThresholdAggFunction field in threshold monitor input
* fix: address failure to terraform worksheets
* feat: add description to preferred path

## 0.8.0 (2022-07-13)
* feat: add default dashboard support for datasets

## 0.7.0 (2022-06-30)
* feat: add dashboard support
* fix: app lookup by name

## 0.6.3 (2022-06-16)

* feat: support managedById for worksheets
* feat: support managedById for monitors

## 0.6.2 (2022-06-09)

* feat: support managedById on datasets
* feat: add useDefaultFreshness to actually apply "freshness"
* ci: use openweather for app test
* feat: implement preferred path

## 0.6.1 (2022-05-23)

* feat: add outputs for app resources
* feat: expose on-demand materialization length
* chore: update dependencies
* ci: remove bash arguments from shebang line
* ci: create Jenkins release job for provider, add support for new S3 path structure
* feat: first iteration of apps

## 0.6.0 (2022-04-19)

* fix: remove `selection` and `selection_value` from monitor data source
* feat: extend http poller
* feat!: deprecate observe_fk
* feat!: deprecate group_by
* feat!: deprecate group_by_columns, group_by_datasets
* feat!: remove notification selection and selection value
* feat: allow workspace lookup by id
* fix: typos in folder error messages

## 0.5.2 (2022-03-23)

* feat: allow specifying body to HTTP poller
* fix: reintroduce terraform caching
* feat: add method to poller
* chore: update deps
* feat: deprecate notification_spec.selection
* feat: support monitor freshness

## 0.5.1 (2022-02-11)

* fix: set source for boards
* feat: add support for observe_folder

## 0.5.0 (2022-02-03)

* fix: don't default GroupByGroups on
* fix: force new on group_by rollback
* fix: handle empty case for group_by_group
* fix: allow rollback from group_by_group
* chore: bump go to 1.17
* fix: ignore group_by_columns if group_by_group is used
* fix: copy pasted typos

## 0.4.21 (2022-01-31)

* feat(monitor): add group_by_group support
* ci: build arm64 binaries

## 0.4.20 (2022-01-19)

* fix(board): typo in error message
* fix: set source for monitor
* testing: reduce verbosity and bump tf version## 0.4.19 (2022-01-19)

## 0.4.19 (2022-01-19)

* feat: add group_by_datasets
* fix: add threshold rule to monitor data source
* chore: vendor update
* feat: add datastream_id for poller

## 0.4.18 (2021-12-03)

* feat: surface datasetid for datastreams

## 0.4.17 (2021-11-19)

* feat: add notify_on_close option to channel action
* feat: add monitor disabled
* vendor: go get -u
* fix: adjust tests

## 0.4.16 (2021-10-28)

* fix: handle resource diff prior to resolution
* chore: update dependencies

## 0.4.15 (2021-10-11)

* ci: bump terraform version
* feat: add observe_worksheet
* feat: allow importing workspace
* feat: allow importing poller
* fix: source dataset test change

## 0.4.14 (2021-09-27)

* feat: data source for datastream
* feat: add datastream_token resource
* feat: add resource support for datastream
* feat: add workspace resource
* feat: add mongodbatlas poller
* fix: lookup workspace, dataset, monitor by name
* fix: rename test workspace

## 0.4.13 (2021-08-27)

* poller: gcp support
* monitors: add threshold monitors
* vendor: update go mod and vendor
* tests: fix broken poller test
* test: make TestAccObserveSourceDatasetResource less flaky

## 0.4.12 (2021-08-03)

* provider: make token and password sensitive
* poller: initial terraform support
* docs: updating docs
* channel: reversing direction of connection between channel and channel action
* sweep: add monitor sweep
* vendor: update go mod and vendor
* tests: test fixes

## 0.4.11 (2021-06-03)

* channel_action: add rate_limit
* vendor: go mod update
* tests: fix regressions
* API: modifying operations now lock

## 0.4.10 (2021-05-10)

* provider: make source_format configurable
* internal: set overwriteSource on saveDataset
* observe_query: add new fields to SnowflakeCursor

## 0.4.9 (2021-04-28)

* observe_dataset: ignore stage foreign keys
* board: suppress JSON whitespace diffs
* monitor: fix schema bug in data source
* dataset: suppress pipeline diff for trailing whitespace
* monitor: add promote rule
* query: allow empty pipeline

## 0.4.8 (2021-04-08)

* makefile: tweak build for local development
* monitor: add monitor data source
* client: surface HTTP error in message
* client: replace github.com/machinebox/graphql
* monitor: deprecate compare_value
* docs: generate docs using tfplugindocs
* dataset: use lastSaved as version

## 0.4.7 (2021-03-10)

* observe_channel: marshal empty slice
* client: add source identifier
* observe_dataset: allow lookup by ID
* internal: remove test panic
* bookmark: fix error message
* observe_board: first pass

## 0.4.6 (2021-02-19)

* observe_source_dataset: bugfix
* observe_monitor: fix facet tests

## 0.4.5 (2021-02-05)

* observe_source_dataset: add IsInsertOnly field to SourceTable resource config
* observe_source_dataset: make source table batch_seq_field optional

## 0.4.4 (2021-02-02)

* observe_monitor: add support for facet
* observe_monitor: fix field name for compare_values
* add preliminary source dataset support
* observe_channel: allow connecting monitors to channels
* add preliminary monitor support
* data_source_query: handle multiple task results
* provider: fix description of customer attribute
* observe_channel_action: remove lastTimeRun

## 0.4.3 (2021-01-05)

* client: deprecate "stageID" field

## 0.4.2 (2020-12-11)

* provider: allow different content-types in http post
* provider: do not `set_id` attribute by omission.
* provider: remove `refresh` attribute from observe_http_post resource
* provider: add assert option to observe_query data source
* vendor: update go-cmp, mapstructure
* provider: add polling to `observe_query`
* provider: add `observe_query` data source
* internal: break out query handling from dataset
* testing: bump tested terraform version to 0.14.2
* testing: test breakage due to Observation schema change

## 0.4.1 (2020-11-13)

* provider: add observe_channel resource
* provider: add observe_channel_action resource type
* provider: add observe_http_post resource
* internal: OB-3402: migrate to MultiStageQueryInput
* perf: use cached client
* internal: fix requiresAuth context accessor name
* testing: cache terraform binary between test runs
* internal: implement collect API, streamline client configs
* internal: propagate context.
* internal: split out API by type.

## 0.4.0 (2020-09-04)

* provider: add description to observe_dataset
* provider: add observe_bookmark resource
* provider: add observe_bookmark_group resource

## 0.3.1 (2020-08-31)

* jenkins: make docker-sign non interactive
* jenkins: remove GPG_TTY, fix upload path.
* jenkins: add ability to sign provider plugins

## 0.3.0 (2020-08-27)

* makefile: modify upload directory structure.
* makefile: set test parallelism
* global: go mod update, bump to 1.15
* client: fix roundtripper return value
* client: add proxy option

## 0.3.0-rc4 (2020-08-02)

* observe_link: ignore dataset version

## 0.3.0-rc3 (2020-07-30)

* provider: add testcase for flag parsing
* provider: add lock for concurrent fk creation
* provider: bugfix for read missing fk
* client: add support for experimental flags.
* provider: add observe_link
* provider: add sweeper test
* client: clean up options
* provider: add test for fk propagation
* client: add retry logic
* provider: add observe_fk data source
* provider: extend observe_dataset data source
* provider: add OBSERVE_INSECURE env var
* provider: export datasets in observe_workspace
* tests: bump terraform version to 0.12.28

## 0.3.0-rc2 (2020-06-30)

* client: add user-agent to all graphql requests.
* vendor: update go mod
* provider: add debug info on lookup failure.
* provider: add support for path_cost
* provider: fix nil pointer dereference if dataset missing.
* provider: request detailedInfo for ResultStatus

