---
title: "redis_pubsub"
---

# redis_pubsub

## Description

The redis filter is used to store received requests and response results to Redis message queues.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: redis_pubsub
    filter:
      - redis_pubsub:
          host: 127.0.0.1
          port: 6379
          channel: gateway
          response: true
```

## Parameter Description

| Name     | Type   | Description                                                              |
| -------- | ------ | ------------------------------------------------------------------------ |
| host     | string | Redis host name, which is `localhost` by default.                        |
| port     | int    | Redis port ID, which is `6379` by default.                               |
| password | string | Redis password                                                           |
| db       | int    | Default database of Redis, which is `0` by default.                      |
| channel  | string | Name of a Redis message queue. It is mandatory and has no default value. |
| response | bool   | Whether the response result is contained. The default value is `true`.   |
