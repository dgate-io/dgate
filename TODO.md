# TODO List

# dgate-cli

- add dgate-cli
  - resource management (CRUD for namespace, domain, service, ...)
  - server management (start-proxy, stop-proxy, restart, status, logs, stats, etc.)
  - cluster management (raft commands, replica commands, etc.) (low priority)
  - other commands (backup, restore, etc.) (low priority)
  - replace k6 with wrk for performance tests

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

At a higher level, background jobs can be used to enable features like health checks, which can periodically check the health of the upstream servers and disable/enable them if they are not healthy.

Other features include: automatic service discovery, ping-based load balancing, 

# Metrics

-- add support for prometheus, datadog, sentry, etc.

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

## Module Permissions (using tags?)

- Allow users to define permissions for modules to access certain dgate resources/apis and/or OS resources.
  - resource:document:read
  - resource:document:write
  - os:net:(http/tcp/udp)
  - os:file:read
  - os:env:read

# Bundles

- Add support for bundles that can be used to extend the functionality of DGate. Bundles are a grouping of resources that can be used to extend the functionality of DGate. Bundles can be used to add new modules, resources, and more.
A good example of a bundle would be a bundle that adds support for OAuth2 authentication. It would need to setup the necessary routes, modules, and configurations to enable OAuth2 authentication. 

## Module/Plugin Variables

- Allow users to define variables that can be used in modules/plugins. These variables can be set by the user, eventually the Admin Console should allow these variables to be set, and the variables can be used in the modules/plugins.

## Mutual TLS Support (low priority)

## Versioning Modules

Differing from common resource versioning, modules can have multiple versions that can be used at the same time. This can be used to test new versions of modules before deploying them to the cluster.


## DGate CLI - argument variable suggestions

For example, if the user types an argument that is not recognized, the CLI can suggest the correct argument by search the available arguments and finding the closest match.
```
dgate-cli ns mk my-ns nmae=my-ns
Variable 'nmae' is not recognized. Did you mean 'name'?
```

## DGate CLI - help command show required variables

When the user runs the help command, the CLI should show the required variables for the command. For example, if the user runs `dgate-cli ns mk --help`, the CLI should show the required variables for the `ns mk` command. `name` is a required variable for the `ns mk` command. Also, the CLI should show non-required variables.

## Improve Module Debugability

Make it easier to debug modules by adding more logging and error handling. This can be done by adding more logging to the modules and making it easier to see the logs in the Admin Console.

Add stack tracing for typescript modules.


## Decouple Admin API from Raft Implementation

Currently, Raft Implementation is tightly coupled with the Admin API. This makes it difficult to change the Raft Implementation without changing the Admin API. Decouple the Raft Implementation from the Admin API to make it easier to change the Raft Implementation.

## Add Telemetry (sentry, datadog, etc.)