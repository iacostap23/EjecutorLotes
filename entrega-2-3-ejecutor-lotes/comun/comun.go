package comun

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Mensaje map[string]interface{}

func Texto(m Mensaje, clave string) string {
	v, ok := m[clave]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func Lista(m Mensaje, clave string) []string {
	v, ok := m[clave]
	if !ok {
		return []string{}
	}
	arreglo, ok := v.([]interface{})
	if !ok {
		return []string{}
	}
	resultado := []string{}
	for _, item := range arreglo {
		if s, ok := item.(string); ok {
			resultado = append(resultado, s)
		}
	}
	return resultado
}

func Ok(campos Mensaje) Mensaje {
	r := Mensaje{"estado": "ok"}
	for k, v := range campos {
		r[k] = v
	}
	return r
}

func Error(mensaje string) Mensaje {
	return Mensaje{"estado": "error", "mensaje": mensaje}
}

func AJson(m Mensaje) string {
	b, err := json.Marshal(m)
	if err != nil {
		return `{"estado":"error","mensaje":"error generando respuesta"}`
	}
	return string(b)
}

// SiguienteID lee el contador del archivo, lo incrementa y devuelve el nuevo ID
func SiguienteID(archivo, prefijo string) (string, error) {
	n := 0
	if b, err := os.ReadFile(archivo); err == nil {
		n, _ = strconv.Atoi(strings.TrimSpace(string(b)))
	}
	n++
	if err := os.WriteFile(archivo, []byte(strconv.Itoa(n)), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%04d", prefijo, n), nil
}
