package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"ejecutor-lotes-go/comun"
	"ejecutor-lotes-go/pipes"
)

type gesfich struct {
	aralmac string
	estado  string // Corriendo | Suspendido | Terminado
}

func main() {
	req := flag.String("f", "lotes-gesfich-req", "tuberia peticiones")
	res := flag.String("b", "lotes-gesfich-res", "tuberia respuestas")
	arc := flag.String("x", "./aralmac", "ruta aralmac")
	flag.Parse()

	g := &gesfich{aralmac: *arc, estado: "Corriendo"}
	os.MkdirAll(filepath.Join(*arc, "ficheros"), 0755)

	pipes.Servidor(*req, *res, func(linea string) (string, bool) {
		var p comun.Mensaje
		if err := json.Unmarshal([]byte(linea), &p); err != nil {
			return comun.AJson(comun.Error("json invalido")), false
		}
		r := g.atender(p)
		return comun.AJson(r), g.estado == "Terminado"
	})
}

func (g *gesfich) atender(p comun.Mensaje) comun.Mensaje {
	if comun.Texto(p, "servicio") != "gesfich" {
		return comun.Error("servicio desconocido")
	}
	op := comun.Texto(p, "operacion")
	if g.estado == "Suspendido" && op != "Leer" && op != "Reasumir" && op != "Terminar" {
		return comun.Error("servicio suspendido")
	}
	switch op {
	case "Crear":
		return g.crear()
	case "Leer":
		return g.leer(p)
	case "Actualizar":
		return g.actualizar(p)
	case "Borrar":
		return g.borrar(p)
	case "Suspender":
		g.estado = "Suspendido"
		return comun.Ok(nil)
	case "Reasumir":
		g.estado = "Corriendo"
		return comun.Ok(nil)
	case "Terminar":
		g.estado = "Terminado"
		return comun.Ok(nil)
	}
	return comun.Error("operacion desconocida")
}

func (g *gesfich) crear() comun.Mensaje {
	id, err := comun.SiguienteID(filepath.Join(g.aralmac, "contador-ficheros.txt"), "f")
	if err != nil {
		return comun.Error("no se pudo crear el fichero")
	}
	if err := os.WriteFile(g.ruta(id), []byte(""), 0644); err != nil {
		return comun.Error("no se pudo crear el fichero")
	}
	return comun.Ok(comun.Mensaje{"id-fichero": id})
}

func (g *gesfich) leer(p comun.Mensaje) comun.Mensaje {
	id := comun.Texto(p, "id-fichero")
	if id == "" {
		// Sin ID: listar todos los ficheros registrados
		entradas, err := os.ReadDir(filepath.Join(g.aralmac, "ficheros"))
		if err != nil {
			return comun.Error("error al listar ficheros")
		}
		ids := []string{}
		for _, e := range entradas {
			if strings.HasSuffix(e.Name(), ".txt") {
				ids = append(ids, strings.TrimSuffix(e.Name(), ".txt"))
			}
		}
		return comun.Ok(comun.Mensaje{"ficheros": ids})
	}
	contenido, err := os.ReadFile(g.ruta(id))
	if err != nil {
		return comun.Error("fichero no encontrado")
	}
	return comun.Ok(comun.Mensaje{"contenido": string(contenido)})
}

func (g *gesfich) actualizar(p comun.Mensaje) comun.Mensaje {
	id, ruta := comun.Texto(p, "id-fichero"), comun.Texto(p, "ruta")
	if id == "" || ruta == "" {
		return comun.Error("faltan campos: id-fichero, ruta")
	}
	if _, err := os.Stat(g.ruta(id)); err != nil {
		return comun.Error("fichero no encontrado")
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return comun.Error("no se pudo actualizar el fichero")
	}
	os.WriteFile(g.ruta(id), contenido, 0644)
	return comun.Ok(nil)
}

func (g *gesfich) borrar(p comun.Mensaje) comun.Mensaje {
	id := comun.Texto(p, "id-fichero")
	if id == "" {
		return comun.Error("faltan campos: id-fichero")
	}
	if err := os.Remove(g.ruta(id)); err != nil {
		return comun.Error("fichero no encontrado")
	}
	return comun.Ok(nil)
}

func (g *gesfich) ruta(id string) string {
	return filepath.Join(g.aralmac, "ficheros", id+".txt")
}
