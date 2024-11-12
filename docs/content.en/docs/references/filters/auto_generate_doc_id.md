---
title: "auto_generate_doc_id"
asciinema: true
---

# auto_generate_doc_id

## Description

The `auto_generate_doc_id` filter is used to add a UUID (Universally Unique Identifier) to a document when creating a document without specifying the UUID explicitly. This is typically used when you don't want the backend system to generate the ID automatically. For example, if you want to replicate the document between clusters, it's better to assign a known ID to the document instead of letting each cluster generate its own ID. Otherwise, this could result in inconsistencies between clusters.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test_auto_generate_doc_id
    filter:
      - auto_generate_doc_id:
```

## Parameter Description

| Name   | Type   | Description                    |
| ------ | ------ | ------------------------------ |
| prefix | string | Add a fixed prefix to the uuid |
