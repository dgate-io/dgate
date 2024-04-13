import {
    addDocument,
    getDocument,
} from "dgate/state";
import { createHash } from "dgate/crypto";

export const requestHandler = async (ctx) => {
    const req = ctx.request();
    const res = ctx.response();
    console.log("req", JSON.stringify(req));
    console.log("res", JSON.stringify(res));
    if (req.method == "GET") {
        if (!req.query.has("id")) {
            res.status(400).json({ error: "id is required" })
            return;
        }
        await getDocument("short_link", req.query.get("id"))
            .then((doc) => {
                console.log("doc", JSON.stringify(doc));
                if (!doc?.data?.url) {
                    res.status(404).json({ error: "not found" });
                } else {
                    console.log("doc", JSON.stringify(doc), req.query.encode());
                    res.redirect(doc.data.url);
                }
            })
            .catch((e) => {
                console.log("error", e, JSON.stringify(e));
                res.status(500).json({ error: e?.message });
            });
        return;
    } else if (req.method == "POST") {
        const hasher = createHash("sha1")
        const link = req.query.get("url");
        console.log("link", link);
        if (!link) {
            res.status(400).json({ error: "link is required" });
        }
        let hash = hasher.update(link).digest("base64rawurl");
        console.log("hash", hash);
        hash = hash.slice(-8);
        console.log("hash", hash);
        return addDocument({
            id: hash,
            collection: "short_link",
            data: {
                url: link,
            }
        }).then(() => {
            res.status(201).json({ id: hash });
        }).catch((e) => {
            res.status(500).json({ error: e?.message });
        });
    } else {
        res.status(405).json({ error: "method not allowed" });
    }
};