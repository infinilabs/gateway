---
title: "sleep"
---

# sleep

## Description

The sleep filter is used to add a fixed delay to requests to reduce the speed.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: slow_query_logging_test
    filter:
      - sleep:
          sleep_in_million_seconds: 1024
```

## Parameter Description

| Name                     | Type  | Description                        |
| ------------------------ | ----- | ---------------------------------- |
| sleep_in_million_seconds | int64 | Delay to be added, in milliseconds |
