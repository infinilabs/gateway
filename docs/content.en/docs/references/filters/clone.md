---
title: "clone"
asciinema: true
---

# clone

## Description

The clone filter is used to clone and forward traffic to another handling flow. It can implement dual-write, multi-write, multi-DC synchronization, cluster upgrade, version switching, and other requirements.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: double_write
    filter:
      - clone:
          flows:
            - write_to_region_a
            - write_to_region_b #last one's response will be output to client
  - name: write_to_region_a
    filter:
      - elasticsearch:
          elasticsearch: es1
  - name: write_to_region_b
    filter:
      - elasticsearch:
          elasticsearch: es2
```

The above example copies Elasticsearch requests to two different remote clusters.

## Parameter Description

| Name     | Type  | Description                                                                                                                                                |
| -------- | ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| flows    | array | Multiple traffic handling flows, which are executed one after another. The result of the last flow is output to the client.                                |
| continue | bool  | Whether to continue the previous flow after traffic is migrated. The gateway returns immediately after it is set to `false`. The default value is `false`. |
