---
title: "queue_consumer"
deprecated: true
---

# queue_consumer

## Description

The queue_consumer processor is used to asynchronously consume requests in a queue and send the requests to Elasticsearch.

## Configuration Example

A simple example is as follows:

```
pipeline:
- name: bulk_request_ingest
  auto_start: true
  keep_running: true
  processor:
    - queue_consumer:
        input_queue: "backup"
        elasticsearch: "backup"
        waiting_after: [ "backup_failure_requests"]
        worker_size: 20
        when:
          cluster_available: [ "backup" ]
```

## Parameter Description

| Name                    | Type   | Description                                                                                                                                                                               |
| ----------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| input_queue             | string | Name of a subscribed queue                                                                                                                                                                |
| worker_size             | int    | Number of threads that concurrently execute consumption tasks, which is set to `1` by default.                                                                                            |
| idle_timeout_in_seconds | int    | Timeout duration of the consumption queue, which is set to `1` by default.                                                                                                                |
| elasticsearch           | string | Name of a target cluster, to which requests are saved.                                                                                                                                    |
| waiting_after           | array  | Data in the main queue can be consumed only after data in a specified queue is consumed.                                                                                                  |
| failure_queue           | string | Request that fails to be executed because of a back-end failure. The default value is `%input_queue%-failure`.                                                                            |
| invalid_queue           | string | Request, for which the returned status code is 4xx. The default value is `%input_queue%-invalid`.                                                                                         |
| compress                | bool   | Whether to compress requests. The default value is `false`.                                                                                                                               |
| safety_parse            | bool   | Whether to enable secure parsing, that is, no buffer is used and memory usage is higher. The default value is `true`.                                                                     |
| doc_buffer_size         | bool   | Maximum document buffer size for the processing of a single request. You are advised to set it to be greater than the maximum size of a single document. The default value is `256*1024`. |
