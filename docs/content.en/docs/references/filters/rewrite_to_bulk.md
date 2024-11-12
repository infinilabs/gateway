---
title: "rewrite_to_bulk"
---

# rewrite_to_bulk

## Description

`rewrite_to_bulk` can analyze ordinary document creation and modification operations in Elasticsearch and rewrite them as Bulk batch requests.

## Configuration Example

Here is a simple example:

```yaml
flow:
   - name: replicate-primary-writes-to-backup-queue
      filter:
        - flow:
            flows:
              - set-auth-for-backup-flow
        - rewrite_to_bulk: # Rewrite docs create/update/delete operation to bulk request
        - bulk_reshuffle: # Handle bulk requests
            when:
              contains:
                _ctx.request.path: /_bulk
            elasticsearch: "backup"
            queue_name_prefix: "async_bulk"
            level: cluster # Cluster, node, index, shard
            partition_size: 10
            fix_null_id: true
        - queue: # Handle non-bulk requests <1. send to non-bulk queue>
            queue_name: "backup"
```

## Parameter Description

| Name                     | Type     | Description                               |
| ------------------------ | -------- | ----------------------------------------- |
| auto_generate_doc_id     | bool     | If it's a create operation and no document ID is specified, whether to auto-generate a document ID, default is `true` |
| prefix                   | string   | Add a fixed prefix to UUID                  |
| type_removed             | bool   | `_type` was removed in latest elasticsearch version, , this option used to remove `_type`  in bulk metadata  |