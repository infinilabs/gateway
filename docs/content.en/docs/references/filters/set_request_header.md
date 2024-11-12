---
title: "set_request_header"
---

# set_request_header

## Description

The set_request_header filter is used to set header information for requests.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: set_request_header
    filter:
      - set_request_header:
          headers:
            - Trial -> true
            - Department -> Engineering
```

## Parameter Description

| Name    | Type | Description                                                           |
| ------- | ---- | --------------------------------------------------------------------- |
| headers | map  | It uses `->` to identify a key value pair and set header information. |
