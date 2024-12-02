---
title: "merge_to_bulk"
asciinema: false
---

# merge_to_bulk

## Description

The merge_to_bulk processor is used to consume pure JSON documents in the queue, and merge them into bulk requests and save them in the specified queue.
It needs to be consumed with the `consumer` processor, and batch writes are used instead of single requests to improve write throughput.

## Configuration Example

A simple example is as follows:

```
pipeline:
  - name: messages_merge_async_bulk_results
    auto_start: true
    keep_running: true
    singleton: true
    processor:
      - consumer:
          queue_selector:
            keys:
              - bulk_result_messages
          consumer:
            group: merge_to_bulk
          processor:
            - merge_to_bulk:
                elasticsearch: "logging"
                index_name: ".infini_async_bulk_results"
                output_queue:
                  name: "merged_async_bulk_results"
                  label:
                    tag: "bulk_logging"
                worker_size: 1
                bulk_size_in_mb: 10
```

## Parameter Description

| 名称                    | 类型   | 说明                                                                                 |
| ----------------------- | ------ | ------------------------------------------------------------------------------------ |
| message_field         | string | The field name in the context where messages from the queue are stored. Default is `messages`. |
| bulk_size_in_kb         | int    | Size of a bulk request, in `KB`.                                                                                                                                                                  |
| bulk_size_in_mb         | int    | Size of a bulk request, in `MB`.                                                                                                                                                                  |
| elasticsearch           | string | Name of a target cluster, to which requests are saved.                                                                                                                                            |
| index_name              | string | Name of the index stored to the target cluster.                                                                                                                                                   |
| type_name               | string | Name of the index type stored to the target cluster. It is set based on the cluster version. The value is `doc` for Elasticsearch versions earlier than v7 and `_doc` for versions later than v7. |
| output_queue.name       | string | The name of output queue                                                                                                                                                                          |
| output_queue.label      | map    | The labels assign to the output queue，with label `type:merge_to_bulk` builtin.                                                                                                                  |
