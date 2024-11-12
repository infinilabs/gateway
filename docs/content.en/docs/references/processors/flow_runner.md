---
title: "flow_runner"
deprecated: true
---

# flow_runner

## Description

The flow_runner processor is used to asynchronously consume requests in a queue by using the processing flow used for online requests.

## Configuration Example

A simple example is as follows:

```
pipeline:
- name: bulk_request_ingest
  auto_start: true
  keep_running: true
  processor:
    - flow_runner:
        input_queue: "primary_deadletter_requests"
        flow: primary-flow-post-processing
        when:
          cluster_available: [ "primary" ]
```

## Parameter Description

| Name          | Type   | Description                                                                                                                                                                                                    |
| ------------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| input_queue   | string | Name of a subscribed queue                                                                                                                                                                                     |
| flow          | string | Flow used to consume requests in consumption queues                                                                                                                                                            |
| commit_on_tag | string | A message is committed only when a specified tag exists in the context of the current request. The default value is blank, indicating that a message is committed immediately after the execution is complete. |
