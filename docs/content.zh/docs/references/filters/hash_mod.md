---
title: "hash_mod"
---

# hash_mod

## 描述

hash_mod 过滤器用来使用请求的上下文通过哈希取模得到一个唯一的分区编号，一般用于后续的请求转发。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: default_flow
    filter:
      - hash_mod: #hash requests to different queues
          source: "$[[_ctx.remote_ip]]_$[[_ctx.request.username]]_$[[_ctx.request.path]]"
          target_context_name: "partition_id"
          mod: 10 #hash to 10 partitions
          add_to_header: true
      - set_context:
          context:
            _ctx.request.header.X-Replicated-ID: $[[_util.increment_id.request_number_id]]_$[[_util.generate_uuid]]
            _ctx.request.header.X-Replicated-Timestamp: $[[_sys.unix_timestamp_of_now]]
            _ctx.request.header.X-Replicated: "true"
```

## 参数说明

| 名称                     | 类型     | 说明                                      |
| ------------------------ | -------- | ----------------------------------------- |
| source                   | string   | 哈希的输入输入，支持变量参数                      |
| target_context_name    | string   | 将分区编号保持到上下文的主键名称 |
| mod                    | int    | 最大分区数        |
| add_to_request_header        | bool     | 是否添加到请求头，默认 `true`，分别为：`X-Partition-ID` 和 `X-Partition-Size`         |
| add_to_response_header        | bool     | 是否添加到响应头，默认 `false`         |