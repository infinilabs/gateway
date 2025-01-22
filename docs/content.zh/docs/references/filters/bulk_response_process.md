---
title: "bulk_response_process"
---

# bulk_response_process

## 描述

bulk_response_process 过滤器用来处理 Elasticsearch 的 Bulk 请求。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: bulk_response_process
    filter:
      - bulk_response_process:
          success_queue: "success_queue"
          tag_on_success: ["commit_message_allowed"]
```

## 参数说明

| 名称                          | 类型   | 说明                                                                        |
| ----------------------------- | ------ | --------------------------------------------------------------------------- |
| invalid_queue                 | string | 保存非法请求的队列名称，必填。                                              |
| failure_queue                 | string | 保存失败请求的队列名称，必填。                                              |
| save_partial_success_requests | bool   | 是否保存 bulk 请求里面部分执行成功的请求，默认 `false`。                    |
| success_queue                 | string | 保存 bulk 请求里面部分执行成功的请求的队列。                                |
| continue_on_error             | bool   | bulk 请求出错之后是否继续执行后面的 filter，默认 `false`                    |
| message_truncate_size         | int    | bulk 请求出错日志截断长度，默认 `1024`                                      |
| safety_parse                  | bool   | 是否采用安全的 bulk 元数据解析方法，默认 `true`                             |
| doc_buffer_size               | int    | 当采用不安全的 bulk 元数据解析方法时，使用的 buffer 大小，默认 `256 * 1024` |
| tag_on_success                | array  | 将所有 bulk 请求处理完成之后，请求上下文打上指定标记                        |
| tag_on_error                  | array  | 请求出现错误的情况下，请求上下文打上指定标记                                |
| tag_on_partial                | array  | 部分请求执行成功的情况下，请求上下文打上指定标记                            |
| tag_on_failure                | array  | 部分请求出现失败（可重试）的情况下，请求上下文打上指定标记                  |
| tag_on_invalid                | array  | 出现不合法请求错误的情况下，请求上下文打上指定标记                          |
| success_flow         | string  |      请求成功执行的 Flow                   |
| invalid_flow         | string  |      非法请求执行的 Flow                   |
| failure_flow         | string  |      失败请求执行的 Flow                   |
