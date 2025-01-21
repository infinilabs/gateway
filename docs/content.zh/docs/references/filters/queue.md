---
title: "queue"
---

# queue

## 描述

queue 过滤器用来保存请求到消息队列。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: queue
    filter:
      - queue: #handle dirty_writes, second-commit
          queue_name: "primary_final_commit_log##$[[partition_id]]"
          labels:
            type: "primary_final_commit_log"
            partition_id: "$[[partition_id]]"
          message: "$[[_ctx.request.header.X-Replicated-ID]]#$[[_ctx.request.header.LAST_PRODUCED_MESSAGE_OFFSET]]#$[[_sys.unix_timestamp_of_now]]"
          when:
            equals:
              _ctx.request.header.X-Replicated: "true"
```

## 参数说明

| 名称            | 类型   | 说明                                     |
| --------------- | ------ | ---------------------------------------- |
| depth_threshold | int    | 大于队列指定深度才能存入队列，默认为 `0` |
| type      | string | 指定消息队列的类型，支持 `kafka` 和 `disk`                             |
| queue_name      | string | 消息队列名称                             |
| labels      | map | 给新增的消息队列 Topic 添加自定义的标签                             |
| message      | string | 自定义消息内容，支持变量                           |
| save_last_produced_message_offset      | bool | 是否保留最后一次写入成功的消息的 Offset 到上下文中，可以作为变量随后使用                           |
| last_produced_message_offset_key      |  string | 自定义最后一次写入成功的消息的 Offset 保留到上下文中的变量名，默认 `LAST_PRODUCED_MESSAGE_OFFSET`                         |
