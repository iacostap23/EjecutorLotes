// main cliente

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"ejecutor-lotes-go/pipes"
)

const MSG_MAX_LEN = 4096

func main() {
	pipePeticion := flag.String("c", "lotes-ctrllt-req", "tuberia de peticiones hacia ctrllt")
	pipeRespuesta := flag.String("a", "lotes-ctrllt-res", "tuberia de respuestas desde ctrllt")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024), MSG_MAX_LEN)

	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())

		if linea == "" {
			continue
		}

		respuesta, err := pipes.Enviar(*pipePeticion, *pipeRespuesta, linea)
		if err != nil {
			fmt.Println(`{"estado":"error","mensaje":"no se pudo conectar con ctrllt"}`)
			continue
		}

		fmt.Println(respuesta)
	}
}
