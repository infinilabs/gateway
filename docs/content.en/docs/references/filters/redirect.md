---
title: "redirect"
---

# redirect

## Description

redirect filter used to redirect request to specify URL address。

## Configuration Example

A simple example is as follows:

```
flow:
  - name: redirect
    filter:
      - redirect:
          uri: https://infinilabs.com
```

## Parameter Description

| Name | Type   | Description                |
| ---- | ------ | -------------------------- |
| uri  | string | The target URI             |
| code | int    | Status code，default `302` |
