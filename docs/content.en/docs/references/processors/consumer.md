---
title: "consumer"
---

# consumer

## Description

The consumer processor is used to consume messages recorded in the `queue` without processing them. Its purpose is to provide an entry point for data consumption pipeline, which will be further processed by subsequent processors.

## Configuration Example

Here is a simple configuration example:

```
pipeline:
  - name: consume_queue_messages
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 5000
    processor:
      - consumer:
          consumer:
            fetch_max_messages: 1
          max_worker_size: 200
          num_of_slices: 1
          idle_timeout_in_seconds: 30
          queue_selector:
            keys:
              - email_messages
          processor:
            - xxx1:
            - xxx2:
```

In the above example, it subscribes to and consumes the `email_messages` queue. The queue messages are stored in the context of the current pipeline. The consumer provides a `processor` parameter, which contains a series of processors that will be executed sequentially. If any processor encounters an error during execution, the `consumer` will exit without committing the batch of data.

## Parameter Description

| Name                  | Type   | Description                                                         |
| --------------------- | ------ | ------------------------------------------------------------------- |
| message_field         | string | The field name in the context where messages from the queue are stored. Default is `messages`. |
| max_worker_size       | int    | The maximum number of workers allowed to run simultaneously. Default is `10`. |
| num_of_slices         | int    | The number of parallel threads for consuming a single queue. Maximum slice size at runtime. |
| slices                | array  | Allowed slice numbers as an integer array.                         |
| queue_selector.labels | map    | Filter a group of queues to be consumed based on labels, similar to `queues` configuration. |
| queue_selector.ids    | array  | Specifies the UUIDs of the queues to be consumed, as a string array. |
| queue_selector.keys   | array  | Specifies the unique key paths of the queues to be consumed, as a string array. |
| queues                | map    | Filter a group of queues to be consumed based on labels, similar to `queue_selector.labels` configuration. |
| waiting_after         | array  | Whether to wait for specified queues to finish consumption before starting. UUIDs of the queues, as a string array. |
| idle_timeout_in_seconds | int  | Timeout duration for consuming queues. Default is `5` seconds. |
| detect_active_queue   | bool   | Whether to automatically detect new queues that meet the conditions. Default is `true`. |
| detect_interval       | int    | Time interval in milliseconds for automatically detecting new queues that meet the conditions. Default is `5000`. |
| quiet_detect_after_idle_in_ms | bool | Idle interval in milliseconds to exit automatic detection. Default is `30000`. |
| skip_empty_queue      | bool   | Whether to skip consuming empty queues. Default is `true`. |
| quit_on_eof_queue     | bool   | Automatically quit consuming when reaching the last message of a queue. Default is `true`. |
| consumer.source       | string | Consumer source.                                                   |
| consumer.id           | string | Unique identifier for the consumer.                                |
| consumer.name         | string | Consumer name.                                                     |
| consumer.group        | string | Consumer group name.                                               |
| consumer.fetch_min_bytes | int  | Minimum size in bytes for fetching messages. Default is `1`.       |
| consumer.fetch_max_bytes | int  | Maximum size in bytes for fetching messages. Default is `10485760`, which is 10MB. |
| consumer.fetch_max_messages | int | Maximum number of messages to fetch. Default is `1`.               |
| consumer.fetch_max_wait_ms | int | Maximum wait time in milliseconds for fetching messages. Default is `10000`. |
| consumer.eof_retry_delay_in_ms | int | Waiting time in milliseconds for retrying when reaching the end of a file. Default is `500`. |