---
title: "index_diff"
---

# index_diff

## 描述

index_diff 处理器用来对两个结果集进行差异对比。

## 配置示例

一个简单的示例如下：

```
pipeline:
- name: bulk_request_ingest
  auto_start: true
  keep_running: true
  processor:
    - index_diff:
        diff_queue: "diff_result"
        buffer_size: 1
        text_report: true #如果要存 es，这个开关关闭，开启 pipeline 的 diff_result_ingest 任务
        source_queue: 'source_docs'
        target_queue: 'target_docs'
```

## 参数说明

| 名称         | 类型   | 说明                                  |
| ------------ | ------ | ------------------------------------- |
| source_queue | string | 来源数据的名称                        |
| target_queue | string | 目标数据的名称                        |
| diff_queue   | string | 存放 diff 结果的队列                  |
| buffer_size  | int    | 内存 buffer 大小                      |
| keep_source  | bool   | diff 结果里面是否包含文档 source 信息 |
| text_report  | bool   | 是否输出文本格式的结果                |
