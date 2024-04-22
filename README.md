# DGate - Distributed API Gateway

[![Go Report Card](https://goreportcard.com/badge/github.com/dgate-io/dgate)](https://goreportcard.com/report/github.com/dgate-io/dgate)
[![Go Reference](https://pkg.go.dev/badge/github.com/dgate-io/dgate.svg)](https://pkg.go.dev/github.com/dgate-io/dgate)
[![CI](https://github.com/dgate-io/dgate/actions/workflows/built_test_bench.yml/badge.svg)](https://github.com/dgate-io/dgate/actions/workflows/ci.yml/badge.svg)
[![E2E](https://github.com/dgate-io/dgate/actions/workflows/built_test_bench.yml/badge.svg)](https://github.com/dgate-io/dgate/actions/workflows/ci.yml/badge.svg)
[![codecov](https://codecov.io/gh/dgate-io/dgate/graph/badge.svg?token=KIDT82HSO9)](https://codecov.io/gh/dgate-io/dgate)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![GitHub Release](https://img.shields.io/github/v/release/dgate-io/dgate)


DGate is a distributed API Gateway built for developers. DGate allows you to use JavaScript/TypeScript to modify request/response data(L7). Inpired by [k6](https://github.com/grafana/k6) and [kong](https://github.com/Kong/kong).

> DGate is currently in development and is not ready for production use. Please use at your own discretion.

## Getting Started

Coming soon @ http://dgate.io/docs/getting-started

### Installing

```bash
# requires go 1.22+
go install github.com/dgate-io/dgate/cmd/dgate-server@latest
```

## Application Architecture

### DGate Server (dgate-server)

DGate Server is proxy and admin server bundled into one. the admin server is responsible for managing the state of the proxy server. The proxy server is responsible for routing requests to upstream servers. The admin server can also be used to manage the state of the cluster using the Raft Consensus Algorithm.

#### Proxy Modules

- Fetch Upstream Module (`fetchUpstream`) - executed before the request is sent to the upstream server. This module is used to decided which upstream server to send the current request to. (Essentially a custom load balancer module)

- Request Modifier Module (`requestModifier`) - executed before the request is sent to the upstream server. This module is used to modify the request before it is sent to the upstream server.

- Response Modifier Module (`responseModifier`) - executed after the response is received from the upstream server. This module is used to modify the response before it is sent to the client.

- Error Handler Module (`errorHandler`) - executed when an error occurs when sending a request to the upstream server. This module is used to modify the response before it is sent to the client.

- Request Handler Module (`requestHandler`) - executed when a request is received from the client. This module is used to handle arbitrary requests, instead of using an upstream service.



- Examples
  - [x] ip hash load balancer
  - [x] short url service
  - [x] modify json request/response
  - [x] send multiple upstream requests and combine them

