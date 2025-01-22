---
title: "auto_generate_doc_id"
asciinema: true
---

# auto_generate_doc_id

## 描述

过滤器 `auto_generate_doc_id` 用于在创建文档时为其添加 UUID（通用唯一标识符），当创建文档时没有显式指定 UUID 时使用该过滤器。通常情况下，这适用于不希望后端系统自动生成 ID 的情况。例如，如果您想在集群之间复制文档，最好为文档分配一个已知的 ID，而不是让每个集群为文档生成自己的 ID。否则，这可能导致集群之间的不一致性。

## 配置示例

A simple example is as follows:

```
flow:
  - name: test_auto_generate_doc_id
    filter:
      - auto_generate_doc_id:
```

## 参数说明

| 名称   | 类型   | 说明                     |
| ------ | ------ | ------------------------ |
| prefix | string | 给 UUID 增加一个固定前缀 |
