---
title: "queue"
---

# queue

## Description

The `queue` filter is used to save requests to a message queue.

## Configuration Example

Here is a simple example:

```yaml
flow:
  - name: queue
    filter:
      - queue: # Handle dirty_writes, second-commit
          queue_name: "primary_final_commit_log##$[[partition_id]]"
          labels:
            type: "primary_final_commit_log"
            partition_id: "$[[partition_id]]"
          message: "$[[_ctx.request.header.X-Replicated-ID]]#$[[_ctx.request.header.LAST_PRODUCED_MESSAGE_OFFSET]]#$[[_sys.unix_timestamp_of_now]]"
          when:
            equals:
              _ctx.request.header.X-Replicated: "true"
```

## Parameter Description

| Name                     | Type     | Description                               |
| ------------------------ | -------- | ----------------------------------------- |
| depth_threshold          | int      | Must be greater than the specified depth to be stored in the queue, default is `0` |
| type                     | string   | Specify the type of message queue, supports `kafka` and `disk` |
| queue_name               | string   | Message queue name                         |
| labels                   | map      | Add custom labels to the newly created message queue topic |
| message                  | string   | Custom message content, supports variables  |
| save_last_produced_message_offset | bool | Whether to retain the Offset of the last successfully written message in the context for later use as a variable |
| last_produced_message_offset_key  | string | Custom variable name for storing the Offset of the last successfully written message in the context, default is `LAST_PRODUCED_MESSAGE_OFFSET` |