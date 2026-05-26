package pipes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

const MsgMaxLen = 4096

type sobreLinux struct {
	Respuesta string `json:"__pipe_respuesta"`
	Mensaje   string `json:"__pipe_mensaje"`
}

func fifo(nombre string) string {
	if strings.HasPrefix(nombre, "/") {
		return nombre
	}

	return "/tmp/" + nombre
}

func crearFIFO(nombre string) error {
	err := syscall.Mkfifo(nombre, 0600)

	if err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

func respuestaTemporal(base string) string {
	base = fifo(base)
	return fmt.Sprintf("%s-%d-%d", base, os.Getpid(), time.Now().UnixNano())
}

func Servidor(pipePeticion, pipeRespuesta string, manejar func(string) (string, bool)) error {
	pipePeticion = fifo(pipePeticion)
	pipeRespuesta = fifo(pipeRespuesta)

	if err := crearFIFO(pipePeticion); err != nil {
		return err
	}

	if err := crearFIFO(pipeRespuesta); err != nil {
		return err
	}

	for {
		entrada, err := os.OpenFile(pipePeticion, os.O_RDONLY, 0600)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(entrada)
		scanner.Buffer(make([]byte, 1024), MsgMaxLen*2)

		for scanner.Scan() {
			linea := strings.TrimSpace(scanner.Text())

			if linea == "" {
				continue
			}

			mensaje := linea
			destinoRespuesta := pipeRespuesta

			var sobre sobreLinux

			if err := json.Unmarshal([]byte(linea), &sobre); err == nil {
				if sobre.Respuesta != "" && sobre.Mensaje != "" {
					mensaje = strings.TrimSpace(sobre.Mensaje)
					destinoRespuesta = fifo(sobre.Respuesta)
				}
			}

			respuesta, terminar := manejar(mensaje)

			salida, err := os.OpenFile(destinoRespuesta, os.O_WRONLY, 0600)
			if err != nil {
				entrada.Close()
				return err
			}

			_, err = fmt.Fprintln(salida, respuesta)
			cerrarErr := salida.Close()

			if err != nil {
				entrada.Close()
				return err
			}

			if cerrarErr != nil {
				entrada.Close()
				return cerrarErr
			}

			if terminar {
				entrada.Close()
				return nil
			}
		}

		err = scanner.Err()
		entrada.Close()

		if err != nil {
			return err
		}
	}
}

func Enviar(pipePeticion, pipeRespuesta, linea string) (string, error) {
	pipePeticion = fifo(pipePeticion)
	pipeRespuestaUnica := respuestaTemporal(pipeRespuesta)

	if err := crearFIFO(pipeRespuestaUnica); err != nil {
		return "", err
	}

	defer os.Remove(pipeRespuestaUnica)

	sobre := sobreLinux{
		Respuesta: pipeRespuestaUnica,
		Mensaje:   linea,
	}

	bytes, err := json.Marshal(sobre)
	if err != nil {
		return "", err
	}

	salida, err := os.OpenFile(pipePeticion, os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}

	_, err = fmt.Fprintln(salida, string(bytes))
	cerrarErr := salida.Close()

	if err != nil {
		return "", err
	}

	if cerrarErr != nil {
		return "", cerrarErr
	}

	entrada, err := os.OpenFile(pipeRespuestaUnica, os.O_RDONLY, 0600)
	if err != nil {
		return "", err
	}

	defer entrada.Close()

	scanner := bufio.NewScanner(entrada)
	scanner.Buffer(make([]byte, 1024), MsgMaxLen)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}

		return "", fmt.Errorf("sin respuesta")
	}

	return scanner.Text(), nil
}
