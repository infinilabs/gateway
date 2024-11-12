---
title: "set_response"
---

# set_response

## Description

The set_response filter is used to set response information to be returned for requests.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: set_response
    filter:
      - set_response:
          status: 200
          content_type: application/json
          body: '{"message":"hello world"}'
```

## Parameter Description

| Name         | Type   | Description                                     |
| ------------ | ------ | ----------------------------------------------- |
| status       | int    | Request status code, which is `200` by default. |
| content_type | string | Type of returned content                        |
| body         | string | Returned structural body                        |
