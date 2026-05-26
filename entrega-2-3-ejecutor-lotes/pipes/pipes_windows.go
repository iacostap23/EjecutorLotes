package pipes

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/Microsoft/go-winio"
)

const MSG_MAX_LEN = 4096

func nombrePipe(nombre string) string {
	if strings.HasPrefix(nombre, `\\.\pipe\`) {
		return nombre
	}

	return `\\.\pipe\` + nombre
}

func Servidor(pipePeticion string, pipeRespuesta string, manejar func(string) (string, bool)) error {
	listener, err := winio.ListenPipe(nombrePipe(pipePeticion), nil)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		terminar := atenderConexion(conn, manejar)
		conn.Close()

		if terminar {
			return nil
		}
	}
}

func atenderConexion(conn net.Conn, manejar func(string) (string, bool)) bool {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024), MSG_MAX_LEN)

	if !scanner.Scan() {
		return false
	}

	linea := strings.TrimSpace(scanner.Text())
	respuesta, terminar := manejar(linea)

	fmt.Fprintln(conn, respuesta)

	return terminar
}

func Enviar(pipePeticion string, pipeRespuesta string, linea string) (string, error) {
	conn, err := winio.DialPipe(nombrePipe(pipePeticion), nil)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = fmt.Fprintln(conn, linea)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024), MSG_MAX_LEN)

	if !scanner.Scan() {
		return "", fmt.Errorf("no hubo respuesta")
	}

	return scanner.Text(), nil
}
