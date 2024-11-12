---
title: "flow"
---

# flow

## Description

The flow filter is used to redirect to or execute one or a series of other flows.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: flow
    filter:
      - flow:
          flows:
          - request_logging
```

Context mapped flow:

```
flow:
  - name: dns-flow
    filter:
      - flow:
          ignore_undefined_flow: true
          context_flow:
            context: _ctx.request.host
            context_parse_pattern: (?P<uuid>^[0-9a-z_\-]+)\.
            flow_id_template: flow_$[[uuid]]
      - set_response:
          status: 503
          content_type: application/json
          body: '{"message":"invalid HOST"}'

```

More information about context, please refer to [Context](../context/) .

## Parameter Description

| Name                               | Type   | Description                                                                                  |
| ---------------------------------- | ------ | -------------------------------------------------------------------------------------------- |
| flow                               | string | Flow ID, the definition of how requests will be executed                                     |
| flows                              | array  | Flow ID, in the array format. You can specify multiple flows, which are executed in sequence |
| ignore_undefined_flow              | bool   | Ignore the undefined flow                                                                    |
| context_flow.context               | string | The context to use for mapping flow_id                                                       |
| context_flow.context_parse_pattern | string | The regexp pattern used to extract named variables from context value                        |
| context_flow.flow_id_template      | string | The string template used to rendering flow_id                                                |
| context_flow.continue              | string | Will continue next filter after executed the context mapped flow, default `false`            |
