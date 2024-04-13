package configtest

const (
	EmptyAsyncModuleFunctionsTS = `
	export const fetchUpstream = async () => {}
	export const requestModifier = async () => {}
	export const responseModifier = async () => {}
	export const errorHandler = async () => {}
	export const requestHandler = async () => {}
	`
)
