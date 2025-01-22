---
title: "rewrite_to_bulk"
---

# rewrite_to_bulk

## 描述

`rewrite_to_bulk` 可以分析 Elasticsearch 的普通文档创建和修改操作并改写为 Bulk 批次请求。

## 配置示例

一个简单的示例如下：

```
flow:
   - name: replicate-primary-writes-to-backup-queue
      filter:
        - flow:
            flows:
              - set-auth-for-backup-flow
        - rewrite_to_bulk: #rewrite docs create/update/delete operation to bulk request
        - bulk_reshuffle: #handle bulk requests
            when:
              contains:
                _ctx.request.path: /_bulk
            elasticsearch: "backup"
            queue_name_prefix: "async_bulk"
            level: cluster #cluster,node,index,shard
            partition_size: 10
            fix_null_id: true
        - queue: #handle none-bulk requests<1. send to none-bulk queue>
            queue_name: "backup"
```

## 参数说明

| 名称                     | 类型     | 说明                                      |
| ------------------------ | -------- | ----------------------------------------- |
| auto_generate_doc_id         | bool | 如果是创建操作，并且没有指定文档 ID，是否自动生成文档 ID，默认 `true`                       |
| prefix | string | 给 UUID 增加一个固定前缀 |
| type_removed             | bool   | 新版本 ES 移除了 `_type` 类型，这个参数用来避免在 Bulk 请求元数据添加类型参数  |