---
weight: 80
title: "Release Notes"
---

# Release Notes

Information about release notes of INFINI Gateway is provided here.

## Latest (In development)

### Breaking changes

### Features

### Bug fix

### Improvements

## 1.28.1 (2025-01-24)

### Features

- Support simple bulk metadata in bulk related filters (#59)
- Seamlessly retry requests on backend failure for the Elasticsearch filter (#63)

### Improvements

- Remove unnecessary node re-selection on schema mismatch for elasticsearch filter (#62)

## 1.28.0 (2025-01-11)

This release includes updates from the underlying [Framework v1.1.0](https://docs.infinilabs.com/framework/v1.1.0/docs/references/http_client/), which resolves several common issues and enhances overall stability and performance. While there are no direct changes to Gateway itself, the improvements inherited from Framework benefit Gateway indirectly.

## 1.27.0 (2024-12-13)

### Breaking Changes

- Update default branch for vendor

### Features

- Auto-issue certificates for domains
- Support filter config file (#620)
- Add configs parameter allow_generated_metrics_tasks
- Add parameter context for ES API ClusterHealth and ClusterStats
- Support passing query parameter level to cluster health API
- Crontab task support for timezones
- Support timezone draft
- Customize event queue

### Bug Fixes

- Rollback for client registration
- Reload when file is in dirty read
- Prevent consumer from advancing beyond writer’s segment
- Fix wait group usage in bulk indexing processor
- Query string query was ignored (#588)

### Improvements

- Refactor system configuration
- Add UUID to WebSocket session
- Add utility to parse parameters, panic on missing
- Adjust logging format
- Refactor inflight check
- Update license header
- Add utility to register schema
- Log with IP and agent register during agent restart
- Update CLI naming style (#589)

## 1.26.1 (2024-08-13)

### Improvements

- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

## 1.26.0 (2024-06-07)

### Improvements

- Add wildcard_domain filter
- Remove security filter and translog_viewer

## 1.25.0 (2024-04-30)

### Improvements

- Add push err and log pop timeout err

### Bug fix

- Fix date_range_precision_tuning work with filter query

## 1.24.0 (2024-04-15)

### Improvements

- Refactoring http client tls config
- Write field routing to bulk metadata when field \_routing exists in scrolled doc

### Bug fix

- Fix(reshuffle filter): make sure queue config always have label `type`
- Fix rotate config usage

## 1.23.0 (2024-03-01)

### Bug fix

- Fix consumer offsets were not reset after deleting instance queues.

## 1.22.0 (2024-01-26)

### Bug fix

- Fix update default EOF max retry times

### Improvements

- Unified version number with INFINI Console
- Optimize bytes pool
- Limit inflight request size
- Add limit to entry

## 1.21.0 (2023-12-29)

### Bug fix

- Fix log error when temp file was missing

## 1.20.0 (2023-12-01)

### Bug fix

- Fix the number of connections not being released and abnormal memory growth caused by Framework Bug

### Improvements

- Add configuration to allow setting fasthttp client parameters

## 1.19.0 (2023-11-16)

### Features

- Add `http` processor
- Add basic-auth based security to API module
- Allow to register self to config manager
- Allow to panic on config error or not

### Bug fix

- Fix `rewrite_to_bulk` the issue with `_type` was missing in newer version
- Fix `rewrite_to_bulk`, support none-index doc operations

### Improvements

- Update test, assert parse result

## 1.18.0 (2023-09-21)

### Breaking changes

- Finally removed `request_body_truncate` and `response_body_truncate` filter

### Features

- Support kafka based replication
- Add `_util.generate_uuid` to request context
- Add `_util.increment_id.BUCKET_NAME` to request context
- Add `singleton` to pipeline config, prevent multiple pipelines running at the same time
- Add a pluggable distributed locker implementation
- Add a general `preference` config for application
- Generalize abstraction for queue, refactoring disk_queue, complete kafka implementation
- Add `merge_to_bulk` processor, retired `indexing_merge` processor
- Add `flow_replay` processor, retired `flow_runner` processor
- Add `replication_correlation` for replication use case
- Add `hash_mod` filter
- Add new parameters to `bulk_response_process` filter
- Add `request_reshuffle` filter
- Add resource limit, allow to set max number of cpus or binding affinity
- Support nested variables in template
- Add `rewrite_to_bulk` filter

### Bug fix

- Fix retry delay was not working in pipeline
- Fix number was not supported in template
- Fix queue selector by labels, if more than one labels specified, they should all match together neither any match should be found

### Improvements

- Lowercase all module names
- Prefetch elasticsearch metadata during start
- Adding a shutdown signal to application scope
- Refactoring queue api, support kafka management
- Add `enabled` to badger module
- Allow to register module/plugin with priority
- Unified queue usage and initialization
- Optimize `bulk_reshuffle` filter's performance, add header `X-Bulk-Reshuffled` to response
- Support to use variables in `queue` filter, allow to output last produced message offset

## 1.17.0 (2023-07-21)

### Features

- Add filter `consumer` to subscribe message from queue.
- Add filter `stmp` to send email messages.

### Improvements

- Auto skip corrupted disk_queue file

## 1.16.0 (2023-06-30)

### Features

- Add filter `security` to support unified identity management through the Console, and supports LDAP authentication login.

## 1.15.0 (2023-06-08)

### Breaking changes

### Features

- Add filter `auto_generate_doc_id` to fill UUID during document creation

### Bug fix

- Fix `floating_ip` freqently switched between active and standby
- Fix `elasticsearch` overwriting upstream `x-forwarded-for` header.
- Fix `queue_consumer` high CPU usage when queue is empty.

### Improvements

## 1.14.0 (2023-05-25)

### Breaking changes

### Features

- Support customize service name
- `metrics` added `user_in_ms` and `sys_in_ms` to track CPU utilization.
- `elasticsearch` added `dial_timeout` option.

### Bug fix

- Fix missing stdout outputs after real-time logging push turned on.
- `logging` fixed broken `min_elapsed_time_in_ms` option.
- Fix a issue that consumes idle queues causing high CPU usage.
  ​

## 1.13.0 (2023-05-11)

### Breaking changes

### Features

- Allow to toggle each rules in router, disabled by default
- Added support for `loong64` architecture.
- Added support for `riscv64` architecture.
- `elasticsearch` added `dial_timeout` option.

### Bug fix

- Fix http response header from upstream was not well returned to client
- Fix duplicated running pipelines when hot-reloading pipeline configruations.
- `bulk_indexing` fix leaking goroutines when pipeline stopped or released.

### Improvements

- Prefer to set headers instead of add headers, avoid generate duplicated headers
- Improve the responsiveness of stopping pipeline.
- `pipeline` added `enabled` option to toggle the pipelines quickly.

## 1.12.1 (2023-04-20)

### Bug fix

- `elasticsearch` fix potential http connection close.
- `elasticsearch` return proper error info when upstream timed out.

## 1.12.0 (2023-04-06)

### Breaking changes

- `bulk_indexing` rename `bulk.response_handle.retry_exception` to `bulk.response_handle.retry_rules`

### Improvements

- `bulk_indexing` improve logging and default retry times.

### Bug fix

- `bulk_indexing` fix potential data loss when 429 status encountered.
- `bulk_indeixng` fix disk queue file retention rules not working properly.
- `metrics` fix prometheus metrics exporting
- `pipeline` fix auto-reloading
- `badger` fix potential file corruption.
- `bulk_reshuffle` fix potential memory leaking.

## 1.11.0

### Breaking changes

- Rename condition `has_fields` to `exists`

### Features

- `echo` added more options: `status`, `continue`, `response`, `logging`, `logging_level`.
- `switch` added `continue` and `unescape` options.
- Add max memory setting to system config
- Add flags to set max memory soft limit

### Bug fix

- Fix path init when install service

### Improvements

- `unescape` path in switch filter by default
- Update badger to v4
- Add more memory info to stats
- Escape metric name for prometheus

## 1.10.0

### Breaking changes

- `bulk_indexing` move bulk response handling controls from `bulk` to `bulk.response_handle`.

### Features

- Support environment variables in config
- Add option `action` to `ratio` filter, allow drop request immediately
- Add `context_parser` filter
- Add `context_switch` filter
- Add `_sys.*` to request context

### Bug fix

### Improvements

## 1.9.0

### Breaking changes

- Refactoring config for ip access control
- Disable elasticsearch metadata refresh by default
- Update default config path from `configs` to `config`
- Remove `sample-configs`, moved to dedicated integrated-testing project
- Remove field `conntime`, update field `@timestamp` to `timestamp` in `logging` filter
- Rename `disorder_` to `fast_`

### Features

- Support listen on IPv6 address
- Add general health api
- Add `request_ip` to context
- Add badger filter plugin
- Allow to split produce and consume messages from s3
- Add `bulk_request_throttle` filter
- Support access request context and more output options in `echo` filter
- Add `body_json` to response context
- Add cert config to API module, support mTLS
- Add api to clear scroll context
- Floating_ip support stick by `priority`
- Add keystore util
- Allow to save success bulk results in `bulk_indexing` processor
- Enable watch and reload the major config file
- Support run background job in one goroutine
- Allow to handle async_bulk request logging
- Add config to control cluster health check while cluster not available, set default to false
- Allow to follow redirects in http filter, set default read and write timeout to 30s
- Support collect instance metrics to monitoring gateway
- Add json log format

### Bug fix

- Fix user was removed in logging filter
- Fix incorrect message size issue, reload when files changed in disk_queue
- Fix issue that `index_diff` could not finished automatically
- Fix hostname was not well updated in filter `set_request_header` or `set_hostname`
- Fix to check consumer's lag instead of queue's lag in `flow_runner` processor
- Fix file not found error for disk_queue
- Fix the delete requests was not proper handled in filter `bulk_reshuffle`, `bulk_request_mutate` and `bulk_indexing` processor
- Fix memory leak caused by misuse of bytes buffer
- Fix to handle the last request in replay processor
- Fix url args was not updated after change
- Fix memory leak when serving high-concurrent requests
- Fix nil id caused error when using sliced workers in `bulk_indexing` processor
- Fix index name with dot
- Refactoring time fields for orm, skip empty time
- Refactoring stats, allow to register extended stats
- Fix to restart gateway entrypoint on flow change
- Update ratio filter, fix random number, add header to ratio filter
- Fix query parameter `no_cache` was not well respected in `get_cache` filter
- Fix single delete request was ignored in bulk requests
- Fix request mutate filter

### Improvements

- Remove newline in indexing_merge and json_indexing processor
- Improve instance check, add config to disable
- Add option `skip_insecure_verify` to s3 module
- Improve instance check, enable config to disable
- Update the way to get ctx process info, optimize memory usage
- Improve indexing performance for `bulk_indexing` processor
- Refactoring disk_queue, speedup message consumption
- Enable segment compress for disk_queue by default
- Skip download s3 files when s3 was not enabled
- Add option to log warning messages for throttle filters
- Optimize hash performance for getting primary shardID and partitionID
- Add cache for get index routing table
- Optimize performance for bulk response processing
- Refactoring bulk_processor, pass meta info to payload func
- Don't call payload func for delete action
- Improve queue consumer's lag check
- Enable prepare flat files ahead for read by default, skip unnecessary file
- Add object pool for xxhash
- Refactoring disk_queue, handle consumer in-flight segments in memory
- Add config to remove duplicated newline for bulk_processor
- Add metric timestamp in stats api
- Improve error on routing table missing
- Refactoring bytes buffer and object pool, expose metrics via API
- Refactoring tasks pooling, support throttle and unified control
- Optimize badger file size and memory usage
- Refactoring time fields for orm, skip empty time
- Refactoring stats, allow to register extended stats
- Refactoring to handle bulk response results
- Add client_session_cache_size to tls setting
- Safety add newline to each bytes when handle bulk requests

## 1.8.1

### Bug fix

- Remove newline in document for processor `es_scroll` and `dump_hash`

## 1.8.0

### Breaking changes

- Remove config `compress_on_message_payload` from disk_queue
- Rename parameter `consumer` to `name` in `consumer_has_lag` condition
- Remove redundancy prefix name of the disk_queue files

### Features

- Add segment level based disk_queue file compression

### Bug fix

- Fix nil host in `bulk_indexing` processor
- Fix nil body in `bulk_response_process` filter
- Fix sliced consume in `bulk_indexing` processor

### Improvements

- Handle bulk stats to `bulk_response_process` and used in `logging` filter

## 1.7.0

### Breaking changes

### Features

- Add prometheus format to `stats` API
- Add `redirect` filter
- Add `context_flow` to `flow` filter
- Add `permitted_client_ip_list` to `router`
- Add Centos based docker image

### Bug fix

- Fix `date_range_precision` filter failed to parse on specify field
- Fix disk usage status in windows platform

### Improvements

- Merge events during config change, prevent unnecessary reload
- Handle templates when loading config
- Add cache to `ldap_auth` filter

## 1.6.0

### Breaking changes

- Update disk_queue folder structure, use UUID as folder name instead of the queue name
- Parameter `mode` was removed from `bulk_reshuffle` filter, only `async` was supported
- Rename filter `bulk_response_validate` to `bulk_response_process`

### Features

- Add metadata to queue
- Support subscribe queue by specify labels
- Support concurrent worker control for `bulk_indexing` processor
- Auto detect new queues for `bulk_indexing` processor
- Allow to consume queue messages over disk queue
- Auto sync disk_queue files to remote s3 in background
- Add api to operate gateway entry
- Support plugin auto discovery
- Add API to operate gateway entities
- Filter `bulk_request_mutate` support remove `_type` in bulk requests for es v8.0+
- Add elasticsearch adapter for version 8.0+
- Add `http` filter for general reverse proxy usage, like proxy Kibana
- Add `consumer_has_lag` condition to check queue status
- Add `record` filter to play requests easier
- Add zstd compress to disk_queue, disabled by default
- Add `disorder_bulk_indexing` processor
- Add `javascript` filter
- Add `prefix` and `suffix` to when conditions
- Add `indexing_merge` processor

### Bug fix

- Fix `date_range_precision_tuning` filter for complex range query
- Fix node availability initially check
- Fix `basic_auth` filter not asking user to input auth info in browser
- Fix `null` id not fixed in filter `bulk_request_mutate` and `bulk_reshuffle`
- Fix `switch` filter not forwarding when `remove_prefix` was disabled
- Fix buffer was not proper reset in `flow_runner` processor
- Fix entry not loading the pre-defined TLS certificates
- Fix `set_basic_auth` not proper reset previous user information
- Fix `elapsed` in request logging not correct
- Fix `switch` filter, use `strings.TrimPrefix` instead of `strings.TrimLeft`
- Fix the last query_string args can't be deleted, parameter `no_cache` in `get_cache` filter fixed
- Fix s3 downloaded file corrupted

### Improvements

- Handle http public address, remove prefix if that exists
- Refactor `bulk_reshuffle` filter and `bulk_indexing` processor
- Should not fetch nodes info when elasticsearch discovery disabled
- Seamless consume queue message across files
- Persist consumer offset to local store
- Add API to reset consumer offset
- Refactoring ORM framework
- Expose error of mapping put
- Refactoring pipeline framework
- Improve multi-instance check, multi-instance disabled by default
- Add CPU and memory metrics to stats api
- Seamless fetch queue files from s3 server
- Proper handle 409 version conflicts in bulk requests
- Allow memory queue to retry 1s when it is full
- Proper handle the cluster available check
- Proper handle the cluster partial failure check
- Exit bulk worker while no new messages returned

## 1.5.0

### Breaking changes

### Features

- Add API to scroll messages from disk queue
- Prevent out of space, disk usage reserved for disk_queue
- Add `context_filter` and `context_limiter` for general purpose
- Add `bulk_request_mutate` filter
- Add `basic_auth` filter
- Add `set_context` filter
- Add `context_regex_replace` filter
- Add `to_string` property to `request` and `response` context

### Bug fix

- Fix bulk response validate incorrectly caused by jsonParser
- Fix nil exception in `request_path_limiter` caused by refactoring
- Fix big size document out of order caused by bulk buffer

### Improvements

- Fix TCP not keepalived in some case
- Add closing progress bar to pipeline module
- Add `retry_delay_in_ms` config to pipeline module
- Handle partial failure in bulk requests
- Optimize scroll performance of `dump_hash` processor
- Improve API directory

## 1.4.0

### Breaking changes

- Rename flow config `filter_v2` to `filter`, only support new syntax
- Rename pipeline config `pipelines_v2` to `pipeline`, `processors` to `processor`, only support new syntax
- Rename filter `request_logging` to `logging`
- Merge dump filters to `dump` filter
- Response headers renamed, dashboard may broken
- Remove filter `request_body_truncate` and `response_body_truncate`

### Features

- Add option to disable file logging output
- Add option `compress` to `queue_consumer` processor

### Bug fix

- Fix invalid host header setting in elasticsearch reverse proxy
- Fix cluster available health check
- Fix gzip encoding issue for requests forwarding

### Improvements

- Support string type in `in` condition

## 1.3.0

### Breaking changes

- Switch to use `pipelines_v2` syntax only
- Rename filter `disk_enqueue` to `queue`
- Rename processor `disk_queue_consumer` to `queue_consumer`
- Rename filter `redis` to `redis_pubsub`

### Features

- Refactoring pipeline framework, support DAG based task schedule
- Add `dump_hash` and `index_diffs` processor
- Add `redis` output and `redis` queue adapter
- Add `set_request_query_args` filter
- Add `ldap_auth` filter
- Add `retry_limiter` filter
- Add `request_body_json_set` and `request_body_json_del` filter
- Add `stats` filter
- Add `health_check` config to `elastic` module
- Add API to pipeline framework, support `_start` and `_stop` pipelines

### Bug fix

- Fix data race issue in bulk_reshuffle
- Fix `fix_null_id` always executed in bulk_reshuffle
- Auto handle big sized documents in bulk requests

### Improvements

- Refactoring flow runner to service pipeline
- Optimize CPU and Memory usage
- Optimize index diff service, speedup and cross version compatibility
- Set the default max file size of queue files to 1 GB
- Proper handle elasticsearch failure during startup
- Support custom depth check to `queue_has_lag` condition
- Support multi hosts for elasticsearch configuration
- Add parameter `auto_start` to prevent pipeline running on start
- Add `keep_running` parameter to pipeline config
- Safety shutdown pipeline and entry service
- Support more complex routing pattern rules

## 1.2.0

### Features

- Support alias in bulk_reshuffle filter.
- Support truncate in request_logging filter.
- Handle 429 retry in json_indexing service.
- Add forcemerge service.
- Add `response_body_regex_replace` filter.
- Add `request_body_regex_replace` filter.
- Add `sleep` filter.
- Add option to log slow requests only.
- Add cluster and bulk status to request logging.
- Add `filter_v2` and support `_ctx` to access request context.
- Add `dump_context` filter.
- Add `translog` filter, support rotation and compression.
- Add `set_response` filter.
- Add `set_request_header` filter.
- Add `set_hostname` filter.
- Add `set_basic_auth` filter.
- Add `set_response_header` filter.
- Add `elasticsearch_health_check` filter.
- Add `drop` filter.

### Bug fix

- Fix truncate body filter, correctly resize the body bytes.
- Fix cache filter.
- Fix floating_ip module.
- Fix dirty write in diskqueue.
- Fix compression enabled requests.
- Fix date_range_precision_tuning filter.
- Fix invalid indices status on closed indices #23.
- Fix document hash for elasticsearch 6.x.
- Fix floating_ip feature run with daemon mode.
- Fix async bulk to work with beats.

### Improvements

- Optimize memory usage, fix memory leak.

### Acknowledgement

#### Thanks to the following enterprises and teams

- China Everbright Bank, China Citic Bank, BSG, Yogoo

#### Thanks to the following individual contributors

- MaQianghua, YangFan, Tanzi, FangLi

## 1.1.0

- Request Logging and Dashboard.
- Support ARM Platform [armv5\v6\v7\v8(arm64)].
- Fix Elasticsearch Nodes Auto Discovery.
- Add Request Header Filter.
- Add Request Method Filter.
- Add Sample Filter.
- Request Logging Performance Optimized (100x speedup).
- Add Request Path Filter.
- Add Debug Filter.
- Add User Info to Logging Message.
- Support Routing Partial Traffic to Specify Processing Flow (by Ratio).
- Support Traffic Clone, Support Dual-Write or 1:N Write.
- Elasticsearch topology auto discovery, support filter by nodes,tags,roles.
- Backend failure auto detection, auto retry and select another available endpoint.
- Floating IP feature ready to use.
- Add bulk_reshuffle filter.

## 1.0.0

- Rewritten for performance
- Index level request throttle
- Request caching
- Kibana MAGIC speedup
- Upstream auto discovery
- Weighted upstream selections
- Max connection limit per upstream
