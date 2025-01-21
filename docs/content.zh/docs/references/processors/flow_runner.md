---
title: "flow_runner"
---

# flow_runner

## 描述

flow_runner 处理器用来异步消费队列里面的请求并使用异步用于在线请求的处理流程来进行消费处理。

## 配置示例

一个简单的示例如下：

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

## 参数说明

| 名称          | 类型   | 说明                                                                               |
| ------------- | ------ | ---------------------------------------------------------------------------------- |
| input_queue   | string | 订阅的队列名称                                                                     |
| flow          | string | 以什么样的流程来消费队列里面的请求消息                                             |
| commit_on_tag | string | 只有当前请求的上下文里面出现指定 tag 才会 commit 消息，默认为空表示执行完就 commit |
