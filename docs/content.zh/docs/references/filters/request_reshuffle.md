---
title: "request_reshuffle"
---

# request_reshuffle

## 描述

`request_reshuffle` 可以分析 Elasticsearch 的非批次请求，归档存储在队列中，通过先落地存储，业务端请求可以快速返回，从而解耦前端写入和后端 Elasticsearch 集群。`request_reshuffle` 需要离线管道消费任务来配合使用。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: backup-flow-request-reshuffle
    filter:
      - flow:
          flows:
            - set-auth-for-backup-flow
      - request_reshuffle: #reshuffle none-bulk requests
          elasticsearch: "backup"
          queue_name_prefix: "request_reshuffle"
          partition_size: $[[env.REQUEST_RESHUFFLE_PARTITION_SIZE]]
          tag_on_success: [ "commit_message_allowed" ]
```

## 参数说明

| 名称                     | 类型     | 说明                                      |
| ------------------------ | -------- | ----------------------------------------- |
| elasticsearch             | string | Elasticsearch 集群实例名称                                                                                     |
| queue_name_prefix         | string | 队列的名称前缀，默认为 `async_bulk` ，默认的 Label `type:request_reshuffle`                        |
| partition_size            | int    | 在 `level` 的基础上，会再次基于文档 `_id` 进行分区，通过此参数可以设置最大的分区大小                           |
| continue_after_reshuffle  | bool   | 执行完 Reshuffle 之后是否继续后续的流程，默认 `false`                                                          |
| tag_on_success            | array  | 将所有 bulk 请求处理完成之后，请求上下文打上指定标记                                                           |