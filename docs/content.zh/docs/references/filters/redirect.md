---
title: "redirect"
---

# redirect

## 描述

redirect 过滤器用来跳转到一个指定的 URL。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: redirect
    filter:
      - redirect:
          uri: https://infinilabs.com
```

## 参数说明

| 名称 | 类型   | 说明                        |
| ---- | ------ | --------------------------- |
| uri  | string | 需要跳转的完整目标 URI 地址 |
| code | int    | 状态码设置，默认 `302`      |
