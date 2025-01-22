---
title: "set_hostname"
---

# set_hostname

## 描述

set_hostname 过滤器用来设置请求 Header 关于要访问的主机或域名信息。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: set_hostname
    filter:
      - set_hostname:
          hostname: api.infini.cloud
```

为避免

## 参数说明

| 名称     | 类型   | 说明     |
| -------- | ------ | -------- |
| hostname | string | 主机信息 |
