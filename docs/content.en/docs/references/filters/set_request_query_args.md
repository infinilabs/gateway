---
title: "set_request_query_args"
---

# set_request_query_args

## Description

The set_request_query_args filter is used to set the QueryString parameter information used for requests.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: set_request_query_args
    filter:
      - set_request_query_args:
          args:
            - size -> 10
```

## Parameter Description

| Name | Type | Description                                                                          |
| ---- | ---- | ------------------------------------------------------------------------------------ |
| args | map  | It uses `->` to identify a key value pair and set QueryString parameter information. |
