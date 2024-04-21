import { createHash } from "dgate/crypto";

export const fetchUpstream = async (ctx) => {
    // Get the hash of the IP address in hex format
    const hasher = createHash("sha1");
    // WARN: This code is vulnerable to IP spoofing attacks, do not use it in production.
    const remoteAddr = ctx.request().headers.get("X-Forwarded-For") || ctx.request().remoteAddress;
    const hash = hasher.update(remoteAddr).digest("hex");
    // turn the hex hash into a number by getting the first 4 characters and converting to a number
    const hexHash = parseInt(hash.substr(0, 4), 16);
    const upstreamUrls = ctx.service()!.urls
    // Use the hash to select an upstream server
    ctx.response().headers.add("X-Hash", hexHash + " / " + upstreamUrls.length);
    ctx.response().headers.add("X-Remote-Address", remoteAddr);
    return upstreamUrls[hexHash % upstreamUrls.length];
};

