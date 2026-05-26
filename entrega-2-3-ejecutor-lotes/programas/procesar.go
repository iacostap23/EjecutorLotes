package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	datos, _ := io.ReadAll(os.Stdin)
	texto := strings.TrimSpace(string(datos))

	if texto == "" {
		fmt.Println("programa ejecutado")
		return
	}

	fmt.Println(strings.ToUpper(texto))
}
