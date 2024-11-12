---
title: "retry_limiter"
---

# retry_limiter

## Description

The retry_limiter filter is used to judge whether the maximum retry count is reached for a request, to avert unlimited retries of a request.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: retry_limiter
    filter:
      - retry_limiter:
          queue_name: "deadlock_messages"
          max_retry_times: 3
```

## Parameter Description

| Name            | Type   | Description                                                                                    |
| --------------- | ------ | ---------------------------------------------------------------------------------------------- |
| max_retry_times | int    | Maximum retry count. The default value is `3`.                                                 |
| queue_name      | string | Name of a message queue, to which messages are output after the maximum retry count is reached |
| tag_on_success  | array  | Specified tag to be attached to request context after retry conditions are triggered           |
