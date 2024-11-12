---
title: "record"
---

# record

## Description

The record filter is used to record requests. Output requests can be copied to the console of Kibana for debugging.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: request_logging
    filter:
      - record:
          stdout: true
          filename: requests.txt
```

Examples of the format of request logs output by the record filter are as follows:

```
GET  /_cluster/state/version,master_node,routing_table,metadata/*

GET  /_alias

GET  /_cluster/health

GET  /_cluster/stats

GET  /_nodes/0NSvaoOGRs2VIeLv3lLpmA/stats
```

## Parameter Description

| Name     | Type   | Description                                                                     |
| -------- | ------ | ------------------------------------------------------------------------------- |
| filename | string | Filename of request logs stored in the data directory                           |
| stdout   | bool   | Whether the terminal also outputs the characters. The default value is `false`. |
