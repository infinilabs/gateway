---
title: "indexing_merge"
deprecated: true
asciinema: false
---

# indexing_merge

## Description

The indexing_merge processor is used to consume pure JSON documents in the queue, and merge them into bulk requests and save them in the specified queue.
It needs to be consumed with the `bulk_indexing` processor, and batch writes are used instead of single requests to improve write throughput.

## Configuration Example

A simple example is as follows:

```
pipeline:
  - name: indexing_merge
    auto_start: true
    keep_running: true
    processor:
      - indexing_merge:
          input_queue: "request_logging"
          elasticsearch: "logging-server"
          index_name: "infini_gateway_requests"
          output_queue:
            name: "gateway_requests"
            label:
              tag: "request_logging"
          worker_size: 1
          bulk_size_in_mb: 10
  - name: logging_requests
    auto_start: true
    keep_running: true
    processor:
      - bulk_indexing:
          bulk:
            compress: true
            batch_size_in_mb: 10
            batch_size_in_docs: 5000
          consumer:
            fetch_max_messages: 100
          queues:
            type: indexing_merge
          when:
            cluster_available: [ "logging-server" ]
```

## Parameter Description

| Name                    | Type   | Description                                                                                                                                                                                       |
| ----------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| input_queue             | int    | Name of a subscribed queue                                                                                                                                                                        |
| worker_size             | int    | Number of threads that concurrently execute consumption tasks, which is set to `1` by default.                                                                                                    |
| idle_timeout_in_seconds | int    | Timeout duration of the consumption queue, in seconds. The default value is `5`.                                                                                                                  |
| bulk_size_in_kb         | int    | Size of a bulk request, in `KB`.                                                                                                                                                                  |
| bulk_size_in_mb         | int    | Size of a bulk request, in `MB`.                                                                                                                                                                  |
| elasticsearch           | string | Name of a target cluster, to which requests are saved.                                                                                                                                            |
| index_name              | string | Name of the index stored to the target cluster.                                                                                                                                                   |
| type_name               | string | Name of the index type stored to the target cluster. It is set based on the cluster version. The value is `doc` for Elasticsearch versions earlier than v7 and `_doc` for versions later than v7. |
| output_queue.name       | string | The name of output queue                                                                                                                                                                          |
| output_queue.label      | map    | The labels assign to the output queue，with label `type:indexing_merge` builtin.                                                                                                                  |
| failure_queue           | string | The name of queue to save failure requests                                                                                                                                                        |
| invalid_queue           | string | The name of queue to save invalid requests                                                                                                                                                        |
