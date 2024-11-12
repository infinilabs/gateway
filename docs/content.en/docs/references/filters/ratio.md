---
title: "ratio"
asciinema: true
---

# ratio

## Description

The ratio filter is used to forward normal traffic to another flow proportionally. It can implement canary release, traffic migration and export, or switch some traffic to clusters of different versions for testing.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: ratio_traffic_forward
    filter:
      - ratio:
          ratio: 0.1
          flow: hello_world
          continue: true
```

## Parameter Description

| Name     | Type   | Description                                                                                                               |
| -------- | ------ | ------------------------------------------------------------------------------------------------------------------------- |
| ratio    | float  | Proportion of traffic to be migrated                                                                                      |
| action   | string | The action when hit, can be `drop` or `redirect_flow`, default is `redirect_flow`                                         |
| flow     | string | New traffic processing flow                                                                                               |
| continue | bool   | Whether to continue flow after hit. Request returns immediately after it is set to `false`. The default value is `false`. |
