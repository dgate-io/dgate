
export const responseModifier = async (ctx: any) => {
    console.log("responseModifier -> path params", ctx.pathParams());
}