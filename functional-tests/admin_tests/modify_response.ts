export const responseModifier = async (ctx) => {
    return fetch("https://httpbin.org/uuid")
        .then(async (res) => {
            const uuidData = await res.json();
            console.log("INFO uuid", JSON.stringify(uuidData));
            const resp = ctx.upstream();
            const results = await resp.getJson();
            results.data.uuid = uuidData.uuid;
            return resp.setStatus(200).setJson(results);
        });
};

export const errorHandler = async (ctx, error) => {
    console.error("ERROR", error);
    ctx.response().status(500).json({ error: error?.message ?? "Unknown error" });
};