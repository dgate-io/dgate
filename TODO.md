# TODO List

# dgate-cli

- add dgate-cli
  - resource management (CRUD for namespace, domain, service, ...)
  - server management (start-proxy, stop-proxy, restart, status, logs, stats, etc.)
  - cluster management (raft commands, replica commands, etc.) (low priority)
  - other commands (backup, restore, etc.) (low priority)
  - replace k6 with wrk for performance tests

## Replace zerolog with slog

## Improve async function performance

There is a pretty significant difference in performance when using async function.


## Add Module Tests

- Test multiple modules being used at the same time
  - [ ] - automatically detect export conflicts
  - [ ] - Add option to specify export variables when ambiguous (?)
  - [ ] - check how global variable conflicts are handled

## Start using Pkl

replace dgate server config with pkl.

## dgate-cli declaritive config

define resources in a declaritive way using pkl. 

## Background Jobs

background jobs can be used to execute code periodically or on specific events. custom events can be triggered from inside modules.

For examples, upstream health checks to make sure that the upstream server is still available.
Also to execute code on specific events, like when a new route is added, or when an http requ.

- event listeners: intercept events and return modified/new data
  - fetch (http requests made by the proxy)
  - request (http requests made to the proxy)
  - resource CRUD operations (namespace/domain/service/module/route/collection/document)
- execute cron jobs: @every 1m, @cron 0 0 * * *, @daily, @weekly, @monthly, @yearly

# Metrics

expose metrics for the following:
- proxy server
  - request count
  - request latency
  - request module latency
  - request upstream latency
  - request error count
- admin server
  - request error latency
  - request count
  - request latency
  - request error count
- modules
  - request count
  - request latency
  - request error count
  - request error latency

- Add Module Permissions Functionality
- Add Change Versioning
- Add Change Rollback
- Add Server Tags
- Add Transactions
  - [ ] - Add transactional support for admin API

## DGate Documentation (dgate.io/docs)

Use Docusaurus to create the documentation for DGate.

## DGate Admin Console (low priority)

Admin Console is a web-based interface that can be used to manage the state of the cluster. Manage resource, view logs, stats, and more. It can also be used to develop and test modules directly in the browser.

## DGate Runtime (low priority)

DGate Runtime is a JavaScript/TypeScript runtime that can be used to test modules. It can be used to test modules before deploying them to the cluster.

## Implement an optimal RuntimePool (low priority)

RuntimePool is a pool of runtimes that can be used to execute modules. It can be used to manage the different modules and clean up resources when they are no longer needed or idle for a certain amount of time.

## TCP Proxy/Gateway (L4LB) (low priority)

Using the same architecture as the HTTP Proxy/Gateway, create a TCP Proxy/Gateway that can be used to proxy TCP connections to upstream servers.

A 'Custom Protocols API'can  allow users to define custom protocols that can be used to proxy TCP connections to upstream servers or handle the connections themselves.

The custom protocols can be defined using JavaScript/TypeScript function or using protocol definitions (API) which will allow these values to be passed to the JavaScript/TypeScript code.

```
{
  "name": "custom_protocol",
  "version": "1",
  "description": "Custom Protocol",
  "modules": ["module_x"]
  "format_definitions": [
    {
      "name": "command",
      "type": "uint8"
    }
    {
      "name": "data_len",
      "type": "int16"
    }
    {
      "name": "data",
      "type": "string",
      "length": "variable.data_len.length"
    }
  ]
}
```

## Server Tags

No special characters are allowed in the tag name or value
- key:value
- value

canary tags
canary@0.01%:canary_mod

time based tags
@(2022-01-01T00:00:00Z)#key:value

## Resource Tags

- name:data
  - no prefix, so this tag is ignored
- #name:data
  - means that the server must have these tags, for the object to be applied
- !name:data
  - means that the server must not have these tags, for the object to be applied
- ?name:data1,data2
  - means that the server must have *any* of these tags, for the object to be applied
- ?*name:data1,data2
  - means that the server must have *one (and only one)* of these tags, for the object to be applied

## Module Permissions

- Allow users to define permissions for modules to access certain dgate resources/apis and/or OS resources.
