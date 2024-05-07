import { sleep } from "dgate";

export const fetchUpstream = async (ctx) =>
    console.debug("fetchUpstream:", JSON.stringify(ctx));

export const requestModifier = async (ctx) =>
    console.debug("requestModifier:", JSON.stringify(ctx));

export const responseModifier = async (ctx) =>
    console.debug("responseModifier:", JSON.stringify(ctx));

export const errorHandler = async (ctx, err) =>
    console.debug("errorHandler:", JSON.stringify(ctx), err);

export const requestHandler = async (ctx) => {
    console.debug("requestHandler:", JSON.stringify(ctx));
    const req = ctx.request()
    if (req.query.has("wait")) {
        const wait = req.query.get("wait")
        await sleep(+wait || wait);
    }
};
