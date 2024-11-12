---
title: "Index Segment Merging"
weight: 30
---

# Active Merging of Index Segments

INFINI Gateway has an index segment merging service, which can actively merge index segment files to improve query speed. The index segment merging service supports sequential processing of multiple indexes and tracks the status of the merging task, thereby preventing cluster slowdown caused by concurrent operations of massive index segment merging tasks.

## Enabling the Service

Modify the `gateway.yml` configuration file by adding the following configuration:

```
force_merge:
  enabled: false
  elasticsearch: dev
  min_num_segments: 20
  max_num_segments: 1
  indices:
    - index_name
```

The parameters are described as follows:

| Name                             | Type   | Description                                                                                          |
| -------------------------------- | ------ | ---------------------------------------------------------------------------------------------------- |
| enabled                          | bool   | Whether the module is enabled, which is set to `false` by default.                                   |
| elasticsearch                    | string | ID of an Elasticsearch cluster, on which index segment merging is performed                          |
| min_num_segments                 | int    | Minimum number of shards in an index for active shard merging. The value is based on indexes.        |
| max_num_segments                 | int    | The maximum number of segment files that can be generated after segment files in a shard are merged  |
| indices                          | array  | List of indexes that need shard merging                                                              |
| discovery                        | object | Auto-discovery of index-related settings                                                             |
| discovery.min_idle_time          | string | Minimum time span for judging whether segment merging conditions are met. The default value is `1d`. |
| discovery.interval               | string | Interval for detecting whether segment merging is required                                           |
| discovery.rules                  | array  | Index matching rules used in automatic index detection                                               |
| discovery.rules.index_pattern    | string | Pattern of indexes that need index segment file merging                                              |
| discovery.rules.timestamp_fields | array  | List of fields representing the index timestamp                                                      |
