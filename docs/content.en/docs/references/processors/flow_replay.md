---
title: "flow_replay"
asciinema: false
---

# flow_replay

## 描述

flow_replay 处理器用来异步消费队列里面的请求并使用异步用于在线请求的处理流程来进行消费处理。

## 配置示例

一个简单的示例如下：

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

## 参数说明

| 名称                    | 类型   | 说明                                                                                 |
| ----------------------- | ------ | ------------------------------------------------------------------------------------ |
| message_field             | string    | 从队列获取到的消息，存放到上下文的字段名称, 默认 `messages`                          |
| flow          | string | 以什么样的流程来消费队列里面的请求消息                                             |
| commit_on_tag | string | 只有当前请求的上下文里面出现指定 tag 才会 commit 消息，默认为空表示执行完就 commit |
