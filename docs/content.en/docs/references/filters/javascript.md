---
title: "javascript"
---

# javascript

## Description

The javascript filter can be used to execute your own processing flow by crafting the scripts in javascript,
which provide the ultimate flexibility.

## Configuration Example

A simple example is as follows:

```
flow:
 - name: test
   filter:
    - javascript:
        source: >
          function process(ctx) {
            var console = require('console');
            console.log("hello from javascript");
          }
```

The `process` in this script is a built-in function that handles incoming context and allows to write your custom business logic.

If the script is complex, it can be loaded from a file:

```
flow:
 - name: test
   filter:
    - javascript:
        file: example.js
```

The `example.js` is where the file located.

## Parameter Description

| Name   | Type   | Description                                                                                                                 |
| ------ | ------ | --------------------------------------------------------------------------------------------------------------------------- |
| source | string | Inline Javascript source code.                                                                                              |
| file   | string | Path to a script file to load. Relative paths are interpreted as relative to the `${INSTANCE_DATA_PATH}/scripts` directory. |
| params | map    | A dictionary of parameters that are passed to the `register` of the script.                                                 |

## Context API

The Context object passed to the process method has the following API. To learn more about context, please refer to [Request Context](../context/).

| Method                   | Description                                                                                                                                                                                                                                                                                          |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Get(string)              | Get a value from the context. If the key does not exist null is returned. If no key is provided then an object containing all fields is returned. eg: `var value = event.Get(key);`                                                                                                                  |
| Put(string, value)       | Put a value into the context. If the key was already set then the previous value is returned. It throws an exception if the key cannot be set because one of the intermediate values is not an object. eg: `var old = event.Put(key, value);`                                                        |
| Rename(string, string)   | Rename a key in the context. The target key must not exist. It returns true if the source key was successfully renamed to the target key. eg: `var success = event.Rename("source", "target");`                                                                                                      |
| Delete(string)           | Delete a field from the context. It returns true on success. eg: `var deleted = event.Delete("user.email");`                                                                                                                                                                                         |
| Tag(string)              | Append a tag to the tags field if the tag does not already exist. Throws an exception if tags exists and is not a string or a list of strings. eg: `event.Tag("user_event");`                                                                                                                        |
| AppendTo(string, string) | AppendTo is a specialized Put method that converts the existing value to an array and appends the value if it does not already exist. If there is an existing value that’s not a string or array of strings then an exception is thrown. eg: `event.AppendTo("error.message", "invalid file hash");` |

## Parameterization

The following example describes how to use `params` to pass variables to scripts that can be loaded from files for easy reuse of program scripts.

```
flow:
 - name: test
   filter:
    - javascript:
        params:
          keyword: [ "hello", "world", "scripts" ]
        source: >
          var console = require('console');
          var params = {keyword: []};
          function register(scriptParams) {
              params = scriptParams;
          }
          function process(ctx) {
            console.info("keyword comes from params: [%s]", params.keyword);
          }
```

`register` is a built-in function that initializes external parameters.
