---
title: "replay"
---

# replay

## 描述

replay 处理器用来重放 `record` 过滤器记录的请求。

## 配置示例

一个简单的示例如下：

```
pipeline:
  - name: play_requests
    auto_start: true
    keep_running: false
    processor:
      - replay:
          filename: requests.txt
          schema: "http"
          host: "localhost:8000"
```

## 参数说明

| 名称     | 类型   | 说明                                   |
| -------- | ------ | -------------------------------------- |
| filename | string | 包含重放消息的文件名称                 |
| schema   | string | 请求协议类型，`http` 或 `https`        |
| host     | string | 接受请求的目标服务器，格式 `host:port` |
