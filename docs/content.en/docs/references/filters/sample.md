---
title: "sample"
asciinema: true
---

# sample

## Description

The sample filter is used to sample normal traffic proportionally. In a massive query scenario, collecting logs of all traffic consumes considerable resources. Therefore, you are advised to perform sampling statistics and sample and analyze query logs.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: sample
    filter:
      - sample:
          ratio: 0.2
```

## Parameter Description

| Name  | Type  | Description    |
| ----- | ----- | -------------- |
| ratio | float | Sampling ratio |
