package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ejecutor-lotes-go/comun"
	"ejecutor-lotes-go/pipes"
)

type gesprog struct {
	aralmac string
	estado  string // Corriendo | Suspendido | Terminado
}

func main() {
	req := flag.String("p", "lotes-gesprog-req", "tuberia peticiones")
	res := flag.String("c", "lotes-gesprog-res", "tuberia respuestas")
	arc := flag.String("x", "./aralmac", "ruta aralmac")
	flag.Parse()

	g := &gesprog{aralmac: *arc, estado: "Corriendo"}

	os.MkdirAll(filepath.Join(*arc, "programas"), 0755)

	pipes.Servidor(*req, *res, func(linea string) (string, bool) {
		var p comun.Mensaje

		if err := json.Unmarshal([]byte(linea), &p); err != nil {
			return comun.AJson(comun.Error("json invalido")), false
		}

		r := g.atender(p)

		return comun.AJson(r), g.estado == "Terminado"
	})
}

func (g *gesprog) atender(p comun.Mensaje) comun.Mensaje {
	if comun.Texto(p, "servicio") != "gesprog" {
		return comun.Error("servicio desconocido")
	}

	op := comun.Texto(p, "operacion")

	if g.estado == "Suspendido" && op != "Leer" && op != "Reasumir" && op != "Terminar" {
		return comun.Error("servicio suspendido")
	}

	switch op {
	case "Guardar":
		return g.guardar(p)

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

func (g *gesprog) guardar(p comun.Mensaje) comun.Mensaje {
	exeOriginal := comun.Texto(p, "ejecutable")

	if exeOriginal == "" {
		return comun.Error("falta campo: ejecutable")
	}

	info, err := os.Stat(exeOriginal)
	if err != nil || info.IsDir() {
		return comun.Error("no se pudo guardar el programa")
	}

	id, err := comun.SiguienteID(filepath.Join(g.aralmac, "contador-programas.txt"), "p")
	if err != nil {
		return comun.Error("no se pudo guardar el programa")
	}

	rutaCopiada, err := g.copiarEjecutable(id, exeOriginal)
	if err != nil {
		return comun.Error("no se pudo guardar el programa")
	}

	meta := comun.Mensaje{
		"id-programa": id,
		"nombre":      filepath.Base(exeOriginal),
		"ejecutable":  rutaCopiada,
		"args":        comun.Lista(p, "args"),
		"env":         comun.Lista(p, "env"),
	}

	b, _ := json.MarshalIndent(meta, "", "  ")

	if err := os.WriteFile(g.ruta(id), b, 0644); err != nil {
		os.RemoveAll(g.rutaDirPrograma(id))
		return comun.Error("no se pudo guardar el programa")
	}

	return comun.Ok(comun.Mensaje{"id-programa": id})
}

func (g *gesprog) leer(p comun.Mensaje) comun.Mensaje {
	id := comun.Texto(p, "id-programa")

	if id == "" {
		entradas, err := os.ReadDir(filepath.Join(g.aralmac, "programas"))
		if err != nil {
			return comun.Error("error al listar programas")
		}

		ids := []string{}

		for _, e := range entradas {
			if strings.HasSuffix(e.Name(), ".json") {
				ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
			}
		}

		return comun.Ok(comun.Mensaje{"programas": ids})
	}

	b, err := os.ReadFile(g.ruta(id))
	if err != nil {
		return comun.Error("programa no encontrado")
	}

	var meta comun.Mensaje
	json.Unmarshal(b, &meta)

	return comun.Ok(comun.Mensaje{"programa": meta})
}

func (g *gesprog) actualizar(p comun.Mensaje) comun.Mensaje {
	id := comun.Texto(p, "id-programa")
	nuevaRuta := comun.Texto(p, "ruta")

	if id == "" || nuevaRuta == "" {
		return comun.Error("faltan campos: id-programa, ruta")
	}

	info, err := os.Stat(nuevaRuta)
	if err != nil || info.IsDir() {
		return comun.Error("no se pudo actualizar el programa")
	}

	b, err := os.ReadFile(g.ruta(id))
	if err != nil {
		return comun.Error("programa no encontrado")
	}

	var meta comun.Mensaje

	if err := json.Unmarshal(b, &meta); err != nil {
		return comun.Error("programa no encontrado")
	}

	rutaCopiada, err := g.copiarEjecutable(id, nuevaRuta)
	if err != nil {
		return comun.Error("no se pudo actualizar el programa")
	}

	meta["ejecutable"] = rutaCopiada
	meta["nombre"] = filepath.Base(nuevaRuta)

	b, _ = json.MarshalIndent(meta, "", "  ")

	if err := os.WriteFile(g.ruta(id), b, 0644); err != nil {
		return comun.Error("no se pudo actualizar el programa")
	}

	return comun.Ok(nil)
}

func (g *gesprog) borrar(p comun.Mensaje) comun.Mensaje {
	id := comun.Texto(p, "id-programa")

	if id == "" {
		return comun.Error("faltan campos: id-programa")
	}

	if _, err := os.Stat(g.ruta(id)); err != nil {
		return comun.Error("programa no encontrado")
	}

	if err := os.Remove(g.ruta(id)); err != nil {
		return comun.Error("programa no encontrado")
	}

	os.RemoveAll(g.rutaDirPrograma(id))

	return comun.Ok(nil)
}

func (g *gesprog) copiarEjecutable(id string, origen string) (string, error) {
	dirPrograma := g.rutaDirPrograma(id)

	if err := os.RemoveAll(dirPrograma); err != nil {
		return "", err
	}

	if err := os.MkdirAll(dirPrograma, 0755); err != nil {
		return "", err
	}

	nombre := filepath.Base(origen)
	destino := filepath.Join(dirPrograma, nombre)

	if err := copiarArchivo(origen, destino); err != nil {
		return "", err
	}

	info, err := os.Stat(origen)
	if err == nil {
		os.Chmod(destino, info.Mode())
	} else {
		os.Chmod(destino, 0755)
	}

	destinoAbs, err := filepath.Abs(destino)
	if err != nil {
		return destino, nil
	}

	return destinoAbs, nil
}

func copiarArchivo(origen string, destino string) error {
	entrada, err := os.Open(origen)
	if err != nil {
		return err
	}
	defer entrada.Close()

	salida, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = io.Copy(salida, entrada)
	cerrarErr := salida.Close()

	if err != nil {
		return err
	}

	return cerrarErr
}

func (g *gesprog) ruta(id string) string {
	return filepath.Join(g.aralmac, "programas", id+".json")
}

func (g *gesprog) rutaDirPrograma(id string) string {
	return filepath.Join(g.aralmac, "programas", id)
}
