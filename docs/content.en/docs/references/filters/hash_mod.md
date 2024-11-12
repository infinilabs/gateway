---
title: "hash_mod"
---

# hash_mod

## Description

The `hash_mod` filter is used to obtain a unique partition number using the hash modulo of the request's context. It is generally used for subsequent request forwarding.

## Configuration Example

A simple example is as follows:

```yaml
flow:
  - name: default_flow
    filter:
      - hash_mod: # Hash requests to different queues
          source: "$[[_ctx.remote_ip]]_$[[_ctx.request.username]]_$[[_ctx.request.path]]"
          target_context_name: "partition_id"
          mod: 10 # Hash to 10 partitions
          add_to_header: true
      - set_context:
          context:
            _ctx.request.header.X-Replicated-ID: $[[_util.increment_id.request_number_id]]_$[[_util.generate_uuid]]
            _ctx.request.header.X-Replicated-Timestamp: $[[_sys.unix_timestamp_of_now]]
            _ctx.request.header.X-Replicated: "true"
```

## Parameter Description

| Name                     | Type     | Description                               |
| ------------------------ | -------- | ----------------------------------------- |
| source                   | string   | Input for the hash, supports variable parameters |
| target_context_name      | string   | The primary key name to store the partition number in the context |
| mod                      | int      | Maximum number of partitions              |
| add_to_request_header    | bool     | Whether to add to the request header, default is `true`, creating `X-Partition-ID` and `X-Partition-Size` headers |
| add_to_response_header   | bool     | Whether to add to the response header, default is `false` |