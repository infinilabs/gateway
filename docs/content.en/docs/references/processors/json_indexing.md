---
title: "json_indexing"
deprecated: true
asciinema: false
---

# json_indexing

## Description

The json_indexing processor is used to consume pure JSON documents in queues and store them to a specified Elasticsearch server.

## Configuration Example

A simple example is as follows:

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
        bulk_size_in_mb: 10
```

## Parameter Description

| Name                    | Type   | Description                                                                                                                                                                                       |
| ----------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| input_queue             | string | Name of a subscribed queue                                                                                                                                                                        |
| worker_size             | int    | Number of threads that concurrently execute consumption tasks, which is set to `1` by default.                                                                                                    |
| idle_timeout_in_seconds | int    | Timeout duration of the consumption queue, in seconds. The default value is `5`.                                                                                                                  |
| bulk_size_in_kb         | int    | Size of a bulk request, in `KB`.                                                                                                                                                                  |
| bulk_size_in_mb         | int    | Size of a bulk request, in `MB`.                                                                                                                                                                  |
| elasticsearch           | string | Name of a target cluster, to which requests are saved.                                                                                                                                            |
| index_name              | string | Name of the index stored to the target cluster.                                                                                                                                                   |
| type_name               | string | Name of the index type stored to the target cluster. It is set based on the cluster version. The value is `doc` for Elasticsearch versions earlier than v7 and `_doc` for versions later than v7. |
