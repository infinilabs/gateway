---
title: "request_body_json_set"
---

# request_body_json_set

## Description

The request_body_json_set filter is used to modify a request body of the JSON format.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - request_body_json_set:
          path:
          - aggs.total_num.terms.field -> "name"
          - aggs.total_num.terms.size -> 3
          - size -> 0
```

## Parameter Description

| Name           | Type | Description                                                                                 |
| -------------- | ---- | ------------------------------------------------------------------------------------------- |
| path           | map  | It uses `->` to identify the key value pair: JSON path and the value used for replacement.  |
| ignore_missing | bool | Whether to ignore processing if the JSON path does not exist. The default value is `false`. |
