---
title: "record"
---

# record

## 描述

record 过滤器是一个记录请求的过滤器，输出的请求可以直接复制到 Kibana 的 Console 中用于调试。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: request_logging
    filter:
      - record:
          stdout: true
          filename: requests.txt
```

record 过滤器输出的请求日志，格式示例如下：

```
GET  /_cluster/state/version,master_node,routing_table,metadata/*

GET  /_alias

GET  /_cluster/health

GET  /_cluster/stats

GET  /_nodes/0NSvaoOGRs2VIeLv3lLpmA/stats
```

## 参数说明

| 名称     | 类型   | 说明                                   |
| -------- | ------ | -------------------------------------- |
| filename | string | 录制请求日志在 data 目录下保存的文件名 |
| stdout   | bool   | 是否在终端也打印输出，默认为 `false`   |
