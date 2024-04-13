
export const fetchUpstream = (ctx) =>
    console.debug("fetchUpstream:", JSON.stringify(ctx));

export const requestModifier = (ctx) =>
    console.debug("requestModifier:", JSON.stringify(ctx));

export const responseModifier = (ctx) =>
    console.debug("responseModifier:", JSON.stringify(ctx));

export const errorHandler = (ctx, err) =>
    console.debug("errorHandler:", JSON.stringify(ctx), err);

export const requestHandler = (ctx) =>
    console.debug("requestHandler:", JSON.stringify(ctx));
