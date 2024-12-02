---
title: "drop"
---

# drop

## Description

The drop filter is used to discard a message and end the processing of a request in advance.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: drop
    filter:
      - drop:
```
