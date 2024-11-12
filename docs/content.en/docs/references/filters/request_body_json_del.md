---
title: "request_body_json_del"
---

# request_body_json_del

## Description

The request_body_json_del filter is used to delete some fields from a request body of the JSON format.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - request_body_json_del:
          path:
          - query.bool.should.[0]
          - query.bool.must
```

## Parameter Description

| Name           | Type  | Description                                                                                 |
| -------------- | ----- | ------------------------------------------------------------------------------------------- |
| path           | array | JSON path key value to be deleted                                                           |
| ignore_missing | bool  | Whether to ignore processing if the JSON path does not exist. The default value is `false`. |
