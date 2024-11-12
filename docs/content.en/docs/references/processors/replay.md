---
title: "replay"
deprecated: true
---

# replay

## Description

The replay processor is used to replay requests recorded by the `record` filter.

## Configuration Example

A simple example is as follows:

```
pipeline:
  - name: play_requests
    auto_start: true
    keep_running: false
    processor:
      - replay:
          filename: requests.txt
          schema: "http"
          host: "localhost:8000"
```

## Parameter Description

| Name     | Type   | Description                                                        |
| -------- | ------ | ------------------------------------------------------------------ |
| filename | string | Name of a file that contains replayed messages                     |
| schema   | string | Request protocol type: `http` or `https`                           |
| host     | string | Target server that receives requests, in the format of `host:port` |
