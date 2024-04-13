# DGate Module Specification

## Module Types

### Request

Request
- functions
    - body()
        - returns the request body
- properties
    - status

```ts
// this keyword has the same type for each module/function, which is the ModuleContext, this contains a snapshot of any metadata information that is available to the module.
interface ModuleContext {
    // this is the request object (requires request permission/minimal)
    request: Request;
    // this is the response object, this may not available for all modules (requires response permission/minimal)
    response: Response;

    // this is the namespace object (requires namespace permission/basic)
    namespace: Namespace;
    // this is the service object (requires service permission/basic)
    service: Service;
    // this is the route object (requires route permission/basic)
    route: Route;
    // this is the module object (requires module permission/basic)
    module: Module;
    
    // this is the node object (requires node permission/advanced)
    node: Node;
    // this is the cluster object (requires cluster permission/advanced)
    cluster: Cluster;
}

function exampleModule(): string {
    const tags = this.node.tags;
    const version = this.node.version;
}
```