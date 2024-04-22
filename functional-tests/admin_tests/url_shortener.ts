// @ts-nocheck
import { createHash } from "dgate/crypto";
import { addDocument, getDocument } from "dgate/state";

export const requestHandler = (ctx: any) => {
    const req = ctx.request();
    const res = ctx.response();
    if (req.method == "GET") {
        if (!req.query.get("id")) {
            res.status(400).json({ error: "id is required" })
            return;
        }
        // get the document with the ID from the collection
        return getDocument("short_link", req.query.get("id"))
            .then((doc: any) => {
                // check if the document contains the URL
                if (!doc?.data?.url) {
                    res.status(404).json({ error: "not found" });
                } else {
                    res.redirect(doc.data.url);
                }
            })
            .catch((e: any) => {
                console.log("error", e, JSON.stringify(e));
                res.status(500).json({ error: e?.message });
            });
    } else if (req.method == "POST") {
        const link = req.query.get("url");
        console.log("link", link);
        if (!link) {
            res.status(400).json({ error: "link is required" });
        }
        // create a hash of the link and use it as the ID
        let hash = createHash("sha1")
            .update(link)
            .digest("base64rawurl")
            .slice(-8);
        // create a new document with the hash as the ID, and the link as the data
        return addDocument({
            id: hash,
            collection: "short_link",
            // the collection schema is defined in url_shortener_test.sh
            data: { url: link },
        }).then(() => {
            res.status(201).json({ id: hash });
        }).catch((e: any) => {
            res.status(500).json({ error: e?.message });
        });
    } else {
        return res.status(405).json({ error: "method not allowed" });
    }
};