---
title: "basic_auth"
---

# basic_auth

## 描述

basic_auth 过滤器用来验证请求的身份认证信息，适用于简单的身份认证。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: basic_auth
    filter:
      - basic_auth:
          valid_users:
            medcl: passwd
            medcl1: abc
            ...
```

## 参数说明

| 名称        | 类型 | 说明         |
| ----------- | ---- | ------------ |
| valid_users | map  | 用户名和密码 |
