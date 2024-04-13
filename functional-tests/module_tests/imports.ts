// Description: Example usage of the dgate library

// fetch - make a request to a remote server (https://fetch.spec.whatwg.org/#fetch-method)
// readWriteBody - reads the body of the request and writes it back. This is useful for modifying the body of a request
// writeResponse - write the custom response
// import { fetch, readWriteBody, writeResponse } from "dgate/http";
// namespaceCache - local server key/value storage for the current namespace
// serviceCache - local server key/value storage for the current service
// requestCache - local server key/value storage for the current request
// import { namespaceCache, serviceCache, routeCache, requestCache } from "dgate/storage";
// useCache(string, opts):[ø,ƒ] - use the cache if it exists
// check(request) - check if the cache exists
// storage(request) - storage the response in the cache and optionally ignore the cache control headers
import { storeRequest, check, useCache } from "dgate/storage";
// fail() - respond with a 500 error
// random(number) - generate a random string of a given length
// sleep(number) - sleep for a number of milliseconds
// import { fail, random, sleep } from "dgate/util";
// callModule - call another module
// checkModule - check if a module exists
// import { callModule, checkModule } from "dgate/modules";
// system - get system information
// import { sleep, fail, retry } from "dgate/util";
// import dgate from "dgate";


export const requestModifier = (res: any, req: any) => {
    res.header.set("Import-Testing");
    // console.log("dgate (JSON.stringify)", JSON.stringify(dgate));
    // retry(4, (i: number) => {
    //     console.log("retry", i);
    //     sleep(0.01);
    //     return i > 3;
    // });
    const [id, setId] = useCache<any>('id', {
        ttl: 10,
        default: (): any => {
            return 0;
        },
        callback: (id: any): any => {
            console.log("callback", id);
            return id + 1;
        }
    });

    console.log("After top");
};
