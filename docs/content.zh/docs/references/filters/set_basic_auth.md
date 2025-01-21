---
title: "set_basic_auth"
---

# set_basic_auth

## 描述

set_basic_auth 过滤器用来设置请求的身份认证信息，可以用于重置请求的身份信息。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: set_basic_auth
    filter:
      - set_basic_auth:
          username: admin
          password: password
```

## 参数说明

| 名称     | 类型   | 说明   |
| -------- | ------ | ------ |
| username | string | 用户名 |
| password | string | 密码   |
