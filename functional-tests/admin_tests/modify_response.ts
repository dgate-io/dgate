export const responseModifier = async (ctx) => {
    return fetch("https://httpbin.org/uuid")
        .then(async (res) => {
            const uuidData = await res.json();
            console.log("INFO uuid", JSON.stringify(uuidData));
            const resp = ctx.upstream();
            const results = await resp.readJson();
            results.data.uuid = uuidData.uuid;
            return resp.status(200).writeJson(results);
        });
};

export const errorHandler = async (ctx, error) => {
    console.error("ERROR", error);
    ctx.response().status(500).json({ error: error?.message ?? "Unknown error" });
};