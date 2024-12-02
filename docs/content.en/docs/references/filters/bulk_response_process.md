---
title: "bulk_response_process"
---

# bulk_response_process

## Description

The bulk_response_process filter is used to process bulk requests of Elasticsearch.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: bulk_response_process
    filter:
      - bulk_response_process:
          success_queue: "success_queue"
          tag_on_success: ["commit_message_allowed"]
```

## Parameter Description

| Name                          | Type   | Description                                                                                                              |
| ----------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------ |
| invalid_queue                 | string | Name of the queue that saves an invalid request. It is mandatory.                                                        |
| failure_queue                 | string | Name of the queue that saves a failed request. It is mandatory.                                                          |
| save_partial_success_requests | bool   | Whether to save partially successful requests in bulk requests. The default value is `false`.                            |
| success_queue                 | string | Queue that saves partially successful requests in the bulk requests                                                      |
| continue_on_error             | bool   | Whether to continue to execute subsequent filters after an error occurs on a bulk request. The default value is `false`. |
| message_truncate_size         | int    | Truncation length of a bulk request error log. The default value is `1024`.                                              |
| safety_parse                  | bool   | Whether to use a secure bulk metadata parsing method. The default value is `true`.                                       |
| doc_buffer_size               | int    | Buffer size when an insecure bulk metadata parsing method is adopted. The default value is `256 * 1024`.                 |
| tag_on_success                | array  | Specified tag to be attached to request context after all bulk requests are processed.                                   |
| tag_on_error                  | array  | Specified tag to be attached to request context after an error occurs on a request.                                      |
| tag_on_partial                | array  | Specified tag to be attached to request context after requests in a bulk request are partially executed successfully.    |
| tag_on_failure                | array  | Specified tag to be attached to request context after some requests in a bulk request fail (retry is supported).         |
| tag_on_invalid                | array  | Specified tag to be attached to request context after an invalid request error occurs.                                   |
| success_flow    | string  | Flow executed upon successful request  |
| invalid_flow    | string  | Flow executed upon invalid request     |
| failure_flow    | string  | Flow executed upon failed request      |
