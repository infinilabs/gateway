---
title: "request_reshuffle"
---

# request_reshuffle

## Description

`request_reshuffle` can analyze non-bulk requests to Elasticsearch, archive them in a queue, and store them on disk first. This allows business-side requests to return quickly, decoupling the front-end writes from the back-end Elasticsearch cluster. `request_reshuffle` requires offline pipeline consumption tasks to work in conjunction.

## Configuration Example

Here is a simple example:

```yaml
flow:
  - name: backup-flow-request-reshuffle
    filter:
      - flow:
          flows:
            - set-auth-for-backup-flow
      - request_reshuffle: # Reshuffle none-bulk requests
          elasticsearch: "backup"
          queue_name_prefix: "request_reshuffle"
          partition_size: $[[env.REQUEST_RESHUFFLE_PARTITION_SIZE]]
          tag_on_success: [ "commit_message_allowed" ]
```

## Parameter Description

| Name                     | Type     | Description                               |
| ------------------------ | -------- | ----------------------------------------- |
| elasticsearch            | string   | Elasticsearch cluster instance name       |
| queue_name_prefix        | string   | Queue name prefix, default is `async_bulk`, default Label is `type:request_reshuffle` |
| partition_size           | int      | In addition to `level`, partitioning is based on the document `_id`. This parameter sets the maximum partition size. |
| continue_after_reshuffle | bool     | Whether to continue with subsequent processes after Reshuffle is complete, default is `false` |
| tag_on_success           | array    | Add specified tags to the request context after processing all bulk requests |