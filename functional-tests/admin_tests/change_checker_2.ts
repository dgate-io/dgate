
export const requestHandler = async (ctx: any) => {
    ctx.response().status(201).json({
        mod: "module2",
    });
};