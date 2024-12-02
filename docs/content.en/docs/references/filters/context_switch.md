---
title: "context_switch"
---

# context_switch

## Description

context_switch filter can be used to use context variables for conditional judgment and achieve flexible jumps.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: context_switch
    filter:
      - context_switch:
          context: logging.month
          default_flow: echo_message_not_found
          switch:
            - case: ["02","01"]
              action: redirect_flow
              flow: echo_message_01_02
            - case: ["03"]
              action: redirect_flow
              flow: echo_message_03
```

## Parameter Description

| Name             | Type     | Description                                                                                                                   |
| ---------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------- |
| context          | string   | The name of context                                                                                                           |
| skip_error       | bool     | Whether to ignore the error and returned directly, such like the context variable does not exist                              |
| default_action   | string   | Set the default action，could be `redirect_flow` or `drop`，default `redirect_flow`                                           |
| default_flow     | string   | Set the default flow                                                                                                          |
| stringify_value  | bool     | Whether to stringify the value，default `true`。                                                                              |
| continue         | bool     | Whether to continue the flow after hit. Request returns immediately after it is set to `false`. The default value is `false`. |
| switch           | array    | Switched by some cases                                                                                                        |
| switch[i].case   | []string | Matched criteria                                                                                                              |
| switch[i].action | string   | The action when met the case，could be `redirect_flow` or `drop`，default `redirect_flow`                                     |
| switch[i].flow   | string   | When action is `redirect_flow`，the flow to redirect，or will use the default flow                                            |
