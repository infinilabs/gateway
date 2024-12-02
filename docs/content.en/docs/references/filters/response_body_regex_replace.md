---
title: "response_body_regex_replace"
---

# response_body_regex_replace

## Description

The response_body_regex_replace filter is used to replace string content in a response by using a regular expression.

## Configuration Example

A simple example is as follows:

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

The result output of the preceding example is `hello world`.

## Parameter Description

| Name    | Type   | Description                                          |
| ------- | ------ | ---------------------------------------------------- |
| pattern | string | Regular expression used for matching and replacement |
| to      | string | Target string used for replacement                   |
