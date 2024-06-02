// @ts-ignore
import { fetch } from "dgate/http";
// @ts-ignore
import { getCache, setCache } from "dgate/storage";

export const requestModifier = async (ctx) => {
    const req = ctx.request();
    // WARN: This code is vulnerable to IP spoofing attacks, do not use it in production.
    const remoteAddr = 
        req.headers.get("X-Forwarded-For") ||
        req.remoteAddress;

    if (!remoteAddr) {
        throw new Error("Failed to get remote address");
    }
    // cache the geodata for 1 hour
    let geodata = getCache('geodata:'+remoteAddr);
    if (!geodata) {
        const georesp = await fetch(`http://ip-api.com/json/${remoteAddr}?fields=192511`);
        if (georesp.status !== 200) {
            throw new Error("Failed to fetch geodata");
        }
        geodata = await georesp.json();
        if (geodata.status === "fail") {
            console.error(JSON.stringify(georesp));
            throw new Error(("IP API: " + geodata?.message) ?? "Failed to fetch geodata");
        }

        // setCache('geodata:'+remoteAddr, geodata, {
        //     ttl: 3600,
        // });
    }

    req.headers.set("X-Geo-Country", geodata.country);
    req.headers.set("X-Geo-CountryCode", geodata.countryCode);
    req.headers.set("X-Geo-Region", geodata.regionName);
    req.headers.set("X-Geo-RegionCode", geodata.region);
    req.headers.set("X-Geo-City", geodata.city);
    req.headers.set("X-Geo-Latitude", geodata.lat);
    req.headers.set("X-Geo-Longitude", geodata.lon);
    req.headers.set("X-Geo-Proxy", geodata.proxy);
    req.headers.set("X-Geo-Postal", geodata.zip);
    req.headers.set("X-Geo-ISP", geodata.isp);
    req.headers.set("X-Geo-AS", geodata.as);
};