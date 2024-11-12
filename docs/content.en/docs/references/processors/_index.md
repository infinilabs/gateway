---
title: "Offline Processor"
weight: 90
bookCollapseSection: true
---

# Pipeline

## What Is Pipeline?

A pipeline is a function combination used for processing tasks offline. It uses the pipeline design pattern, just as online request filters do. A processor is the basic unit of a pipeline.
Each processing component focuses on one task and the components can be flexibly assembled, and plugged and removed as required.

## Pipeline Definition

A typical pipeline service is defined as follows:

```
pipeline:
- name: request_logging_index
  auto_start: true
  keep_running: true
  processor:
    - json_indexing:
        index_name: "gateway_requests"
        elasticsearch: "dev"
        input_queue: "request_logging"
        idle_timeout_in_seconds: 1
        worker_size: 1
        bulk_size_in_mb: 10 #in MB
```

In the above configuration, a processing pipeline named `request_logging_index` is defined, and the `processor` parameter defines several processing units for the pipeline, which are executed in sequence.

## Parameter Description

Parameters related to pipeline definition are described as follows:

| Name              | Type   | Description                                                                                                           |
| ----------------- | ------ | --------------------------------------------------------------------------------------------------------------------- |
| name              | string | Name of a pipeline, which must be unique                                                                              |
| auto_start        | bool   | Whether the pipeline automatically starts with the gateway startup, that is, whether the task is executed immediately |
| keep_running      | bool   | Whether the gateway starts executing the task again after completing the execution                                    |
| singleton        | bool   | Whether the task is a singleton and only one node instance is allowed to run in a cluster     |
| max_running_in_ms        | int   | The maximum time that the task runs execution, `60000` milliseconds by default.     |
| retry_delay_in_ms | int    | Minimum waiting time for the task re-execution, which is set to `5000` milliseconds by default                        |
| processor         | array  | List of processors to be executed by the pipeline in sequence                                                         |

## Processor List

### Task Scheduling

- [dag](./dag)

### Event Processing

- [consumer](./consumer)
- [smtp](./smtp)
- [merge_to_bulk](./merge_to_bulk)
- [flow_replay](./flow_replay)
- [replication_correlation](./replication_correlation)

### Index Writing

- [bulk_indexing](./bulk_indexing)
- [json_indexing](./json_indexing)

### Index Diff

- [dump_hash](./dump_hash)
- [index_diff](./index_diff)

### Request Replay

- [replay](./replay)
