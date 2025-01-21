---
title: "response_body_regex_replace"
---

# response_body_regex_replace

## 描述

response_body_regex_replace 过滤器使用正则表达式来替换请求响应内容的字符串。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - echo:
          message: "hello infini\n"
      - response_body_regex_replace:
          pattern: infini
          to: world
```

上面的结果输出为 `hello world`。

## 参数说明

| 名称    | 类型   | 说明                     |
| ------- | ------ | ------------------------ |
| pattern | string | 用于匹配替换的正则表达式 |
| to      | string | 替换为目标的字符串内容   |
