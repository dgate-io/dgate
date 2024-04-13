export const requestHandler1 = (res: any, req: any) => {
  res.header.add("Import-Testing", "true");
  res.header.set("Content-Type", "text/plain");
  res.header.remove("Via");
  res.status(200).send("Hello World");
};

export const requestHandler2 = (res: any, req: any) => {
  // 301: res.redirectPermanent("https://www.google.com");
  res.redirect("https://www.google.com");
};

export const requestHandler3 = (res: any, req: any) => {
  // res.sendBuffer(Buffer.from("Hello World"));
};

export const requestHandler4 = (res: any, req: any) => {
  res.redirect("https://www.google.com");
};