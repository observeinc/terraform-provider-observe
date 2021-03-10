## 0.4.7 (2021-03-10)

observe_channel: marshal empty slice
client: add source identifier
observe_dataset: allow lookup by ID
internal: remove test panic
bookmark: fix error message
observe_board: first pass

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

