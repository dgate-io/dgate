import { fetch } from "dgate/http";

export const requestModifier = async (ctx) => {
    // WARN: This code is vulnerable to IP spoofing attacks, do not use it in production.
    const remoteAddr = ctx.request().headers.get("X-Forwarded-For")
        || ctx.request().remoteAddress;
    if (!remoteAddr) {
        throw new Error("Failed to get remote address");
    }
    // cache the geodata for 1 hour
    // const [geodata, setGeodata] = useCache<any>('geodata:'+remoteAddr, { ttl: 3600 });
 
    const georesp = await fetch(`http://ip-api.com/json/${remoteAddr}?fields=192511`);
    if (georesp.status !== 200) {
        throw new Error("Failed to fetch geodata");
    }
    const geodata = await georesp.json();
    if (geodata.status === "fail") {
        console.error(JSON.stringify(georesp));
        throw new Error(("IP API: " + geodata?.message) ?? "Failed to fetch geodata");
    }

    const headers = ctx.request().headers;
    headers.set("X-Geo-Country", geodata.country);
    headers.set("X-Geo-CountryCode", geodata.countryCode);
    headers.set("X-Geo-Region", geodata.regionName);
    headers.set("X-Geo-RegionCode", geodata.region);
    headers.set("X-Geo-City", geodata.city);
    headers.set("X-Geo-Latitude", geodata.lat);
    headers.set("X-Geo-Longitude", geodata.lon);
    headers.set("X-Geo-Proxy", geodata.proxy);
    headers.set("X-Geo-Postal", geodata.zip);
    headers.set("X-Geo-ISP", geodata.isp);
    headers.set("X-Geo-AS", geodata.as);
};