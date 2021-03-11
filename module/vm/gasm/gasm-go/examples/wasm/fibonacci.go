//go:generate tinygo build -opt=s -o fibonacci.wasm -target wasm fibonacci.go
package main

func main() {}

//export fibonacci
func fibonacci(in uint32) uint32 {
	if in <= 1 {
		return in
	}
	return fibonacci(in-1) + fibonacci(in-2)
}
