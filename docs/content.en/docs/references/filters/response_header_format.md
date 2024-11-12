---
title: "response_header_format"
---

# response_header_format

## Description

The response_header_format filter is used to convert keys in response header information into lowercase letters.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - response_header_format:
```
