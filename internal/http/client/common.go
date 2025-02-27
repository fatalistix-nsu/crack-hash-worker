package client

const basePath = "/internal/api/worker/hash/crack"

func makeUrl(address, path string) string {
	return "http://" + address + path
}
