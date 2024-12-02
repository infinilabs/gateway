---
title: "set_basic_auth"
---

# set_basic_auth

## Description

The set_basic_auth filter is used to configure the authentication information used for requests. You can use the filter to reset the authentication information used for requests.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: set_basic_auth
    filter:
      - set_basic_auth:
          username: admin
          password: password
```

## Parameter Description

| Name     | Type   | Description |
| -------- | ------ | ----------- |
| username | string | Username    |
| password | string | Password    |
