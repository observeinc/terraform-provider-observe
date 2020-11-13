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

