---
title: "flow_replay"
asciinema: false
---

# flow_replay

## Description

The flow_replay processor is used to asynchronously consume requests in the queue and use the asynchronous processing process for online requests to perform consumption processing.

## Configuration Example

A simple example is as follows:

```
pipeline:
  - name: backup-flow-request-reshuffle
        auto_start: true
        keep_running: true
        singleton: true
        retry_delay_in_ms: 10
        processor:
          - consumer:
              max_worker_size: 100
              queue_selector:
                labels:
                  type: "primary_write_ahead_log"
              consumer:
                group: request-reshuffle
                fetch_max_messages: 10000
                fetch_max_bytes: 20485760
                fetch_max_wait_ms: 10000
              processor:
                - flow_replay:
                    flow: backup-flow-request-reshuffle
                    commit_on_tag: "commit_message_allowed"
```

## Parameter Description

| Name                    | Type   | Description                                                                                 |
| ----------------------- | ------ | ------------------------------------------------------------------------------------ |
| message_field             | string    | The context field name that store the message obtained from the queue, default `messages`.                          |
| flow          | string | Specify the flow to consume request messages in the queue.                                             |
| commit_on_tag | string | Only when the specified tag appears in the context of the current request will the message be committed. The default is empty, which means the commit will be executed once completed.
 |
