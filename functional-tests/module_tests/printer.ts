import { sleep } from "dgate";
import { fetch } from "dgate/http";
import { useCache } from "dgate/storage";
let i = 0;

export const testAsync = async (host: string) => {
    // sleep(1);
    // throw new Error("DGate: test async");
    return host
}

let fetchUpstream = async (svc: any, route: any, param: any): Promise<string> => {
    i += 1;
    console.log(
        useCache<any>('svc', { initialValue: svc })
    );
    console.log("fetchUpstream.svc.name", svc.name);
    svc.name = "test"
    console.log("fetchUpstream.svc.name", svc.name);
    console.log("fetchUpstream.param", JSON.stringify(param));
    console.log("fetchUpstream.svc", JSON.stringify(svc));
    return await testAsync(svc.urls[0]);
};

type Route = {
    name: string;
    path: string;
    method: string;
    service: string;
    domain: string;
    namespace: string;
    request: any;
    response: any;
    error: any;
    requestModifier: any;
    responseModifier: any;
    upstream: any;
    timeout: number;
    retry: any;
    cache: any;
    cors: any;
    security: any;
    metrics: any;
    tracing: any;
    logging: any;
    circuitBreaker: any;
    loadBalancing: any;
    rateLimiting: any;
    authentication: any;
    authorization: any;
    transformation: any;
    validation: any;
    monitoring: any;
    alerting: any;
    testing: any;
    documentation: any;
    custom: any;
}

type DGateContext = {
    route: Readonly<any>;
    service?: Readonly<any>;
    domain: Readonly<any>;
    namespace: Readonly<any>;

    pathParams: Readonly<Map<string, string>>;
    domainParams: Readonly<Map<string, string>>;

    request: Readonly<Request>;
    response: Readonly<Response>;

    error?: Readonly<any>;
    
};

const requestModifier = async (ctx: DGateContext, req: any, resp: any) => {
    // ctx.response.type
    // Response
    // DGate
    // requestContext
    // routeContext
    // serviceContext
    // namespaceContext

    const [svc,] = useCache<any>('svc');
    // console.log("svc", JSON.stringify(svc));
    // console.log("path", req.url.path);
    const [id, setId] = useCache<number>('id', {
        initialValue: 0,
        reducer: (state: number, action: number) => {
            // console.log("state", state);
            // console.log("action", action);
            return state + action;
        }
    });
    console.log("---", id);
    setId(1);
    const [_id, _] = useCache<any>('id', {
        defaultValue: 1,
    });
    // console.log("id", _id);
    i += 1;
    // console.log("requestModifier.i", i);
    // console.log("requestModifier.req", JSON.stringify(req));
    if ("/stats") {
        // console.log("sleeping hi");
        // console.log("fetch", fetch);
        sleep(0.01)
        // const resp = await fetch("https://jsonplaceholder.typicode.com/todos/22", {
        //     method: "GET",
        //     headers: {
        //         "Content-Type": "application/json",
        //     },
        //     // TODO: Implement retry
        //     // retry: {
        //     //     attempts: 3,
        //     //     delay: 1000,
        //     //     maxDelay: 5000,
        //     //     statusCodes: [500, 502, 503, 504],
        //     // },
        // }).catch((err) => {
        //     console.log("resp err", err);
        // });
        // console.log("resp", JSON.stringify(resp), Object.keys(Math), 1);
    }
    // console.log(sleep, "sleep")
    console.log("requestModifier.req", JSON.stringify(req));
    // console.log("requestModifier.upstream", upstream);
    // console.log("requestModifier.upstream", JSON.stringify(upstream));
};

let responseModifier = (res: any, req: any) => {
    console.log("responseModifier.req", JSON.stringify(req));
    console.log("responseModifier.res", JSON.stringify(res));
    res.header.set("Module", "Printer");
    // i += 1;
    // console.log("responseModifier.i", i);
    // console.log("responseModifier.res", JSON.stringify(res));
    // console.log("responseModifier.req", JSON.stringify(req));
};

export {
    requestModifier,
    responseModifier,
    fetchUpstream,
}