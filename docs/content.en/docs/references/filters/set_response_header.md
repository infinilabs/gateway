---
title: "set_response_header"
---

# set_response_header

## Description

The set_response_header filter is used to set the header information used in responses.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: set_response_header
    filter:
      - set_response_header:
          headers:
            - Trial -> true
            - Department -> Engineering
```

## Parameter Description

| Name    | Type | Description                                                           |
| ------- | ---- | --------------------------------------------------------------------- |
| headers | map  | It uses `->` to identify a key value pair and set header information. |
