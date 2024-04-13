import { fetch } from "dgate/http";

export const requestHandler = async (ctx) => {
    await Promise.allSettled([
        fetch("https://httpbin.org/uuid"),
        fetch("https://httpbin.org/headers"),
        fetch("https://httpbin.org/user-agent"),
    ]).then(async (results) => {
        let baseObject = {};
        for (const result of results) {
            if (result.status === "fulfilled") {
                const jsonResults = await result.value.json()
                console.log("INFO time", result.value._debug_time);
                baseObject = {...baseObject, ...jsonResults};
            }
        }
        console.log("INFO fetch", JSON.stringify(baseObject));
        return ctx.response().status(200).json(baseObject);
    });
};