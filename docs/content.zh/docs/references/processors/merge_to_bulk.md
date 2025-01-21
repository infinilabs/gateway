---
title: "merge_to_bulk"
asciinema: false
---

# merge_to_bulk

## 描述

merge_to_bulk 处理器用来消费队列里面的纯 JSON 文档，并合并成 Bulk 请求保存到指定的队列里面，需要配合 `consumer` 处理器进行消费，用批量写入代替单次请求来提高写入吞吐。

## 配置示例

一个简单的示例如下：

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

## 参数说明

| 名称                    | 类型   | 说明                                                                                 |
| ----------------------- | ------ | ------------------------------------------------------------------------------------ |
| message_field             | string    | 从队列获取到的消息，存放到上下文的字段名称, 默认 `messages`                          |
| bulk_size_in_kb         | int    | 批次请求的单位大小，单位 `KB`                                                        |
| bulk_size_in_mb         | int    | 批次请求的单位大小，单位 `MB`，默认 `10`                                             |
| elasticsearch           | string | 保存到目标集群的名称                                                                 |
| index_name              | string | 保存到目标集群的索引名称                                                             |
| type_name               | string | 保存到目标集群的索引类型名称，默认根据集群版本来设置，v7 以前为 `doc`，之后为 `_doc` |
| output_queue.name       | string | 保存到目标队列的名称                                                                 |
| output_queue.label      | map    | 保存到目标队列的标签，内置 `type:merge_to_bulk`                                     |
