package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"ejecutor-lotes-go/comun"
	"ejecutor-lotes-go/pipes"
)

type proceso struct {
	id     string
	progID string
	estado string // Ejecutando | Suspendido | Terminado
	codigo *int
	cmd    *exec.Cmd
}

type ejecutor struct {
	aralmac  string
	estado   string // Corriendo | Suspendido | Parar | Terminado
	contador int
	procesos map[string]*proceso
	mu       sync.Mutex
}

func main() {
	req := flag.String("e", "lotes-ejecutor-req", "tuberia peticiones")
	res := flag.String("d", "lotes-ejecutor-res", "tuberia respuestas")
	arc := flag.String("x", "./aralmac", "ruta aralmac")
	flag.Parse()

	e := &ejecutor{
		aralmac:  *arc,
		estado:   "Corriendo",
		procesos: map[string]*proceso{},
	}

	err := pipes.Servidor(*req, *res, func(linea string) (string, bool) {
		var p comun.Mensaje

		if err := json.Unmarshal([]byte(linea), &p); err != nil {
			return comun.AJson(comun.Error("json invalido")), false
		}

		r := e.atender(p)

		e.mu.Lock()
		terminar := e.estado == "Terminado"
		e.mu.Unlock()

		return comun.AJson(r), terminar
	})

	if err != nil {
		fmt.Println(comun.AJson(comun.Error("error en tuberia")))
	}
}

func (e *ejecutor) atender(p comun.Mensaje) comun.Mensaje {
	if comun.Texto(p, "servicio") != "ejecutor" {
		return comun.Error("servicio desconocido")
	}

	op := comun.Texto(p, "operacion")

	e.mu.Lock()
	estadoActual := e.estado
	e.mu.Unlock()

	if op == "Ejecutar" && estadoActual == "Suspendido" {
		return comun.Error("servicio suspendido")
	}

	if op == "Ejecutar" && estadoActual == "Parar" {
		return comun.Error("servicio parando")
	}

	switch op {
	case "Ejecutar":
		return e.ejecutar(p)

	case "Estado":
		return e.estado_(p)

	case "Matar":
		return e.matar(p)

	case "Suspender":
		return e.suspender()

	case "Reasumir":
		return e.reasumir()

	case "Parar":
		return e.parar()

	default:
		return comun.Error("operacion desconocida")
	}
}

func (e *ejecutor) ejecutar(p comun.Mensaje) comun.Mensaje {
	idProg := comun.Texto(p, "id-programa")

	if idProg == "" {
		return comun.Error("falta campo: id-programa")
	}

	rutaPrograma := filepath.Join(e.aralmac, "programas", idProg+".json")

	b, err := os.ReadFile(rutaPrograma)
	if err != nil {
		return comun.Error("programa no encontrado")
	}

	var meta comun.Mensaje

	if err := json.Unmarshal(b, &meta); err != nil {
		return comun.Error("programa no encontrado")
	}

	ejecutable := comun.Texto(meta, "ejecutable")
	args := comun.Lista(meta, "args")
	env := comun.Lista(meta, "env")

	cmd := exec.Command(ejecutable, args...)

	cmd.Env = append(os.Environ(), env...)

	archivos := []*os.File{}

	stdin, err := abrirFichero(e.aralmac, comun.Texto(p, "stdin"), false)
	if err != nil {
		return comun.Error("fichero stdin no encontrado")
	}

	if stdin != nil {
		cmd.Stdin = stdin
		archivos = append(archivos, stdin)
	}

	stdout, err := abrirFichero(e.aralmac, comun.Texto(p, "stdout"), true)
	if err != nil {
		cerrarArchivos(archivos)
		return comun.Error("fichero stdout no encontrado")
	}

	if stdout != nil {
		cmd.Stdout = stdout
		archivos = append(archivos, stdout)
	}

	stderr, err := abrirFichero(e.aralmac, comun.Texto(p, "stderr"), true)
	if err != nil {
		cerrarArchivos(archivos)
		return comun.Error("fichero stderr no encontrado")
	}

	if stderr != nil {
		cmd.Stderr = stderr
		archivos = append(archivos, stderr)
	}

	if err := cmd.Start(); err != nil {
		cerrarArchivos(archivos)
		return comun.Error("no se pudo ejecutar el programa")
	}

	e.mu.Lock()
	e.contador++
	id := fmt.Sprintf("e-%04d", e.contador)

	proc := &proceso{
		id:     id,
		progID: idProg,
		estado: "Ejecutando",
		cmd:    cmd,
	}

	e.procesos[id] = proc
	e.mu.Unlock()

	go e.esperarProceso(proc, archivos)

	return comun.Ok(comun.Mensaje{
		"id-ejecucion": id,
	})
}

func (e *ejecutor) esperarProceso(proc *proceso, archivos []*os.File) {
	err := proc.cmd.Wait()

	codigo := 0

	if proc.cmd.ProcessState != nil {
		codigo = proc.cmd.ProcessState.ExitCode()
	} else if err != nil {
		codigo = 1
	}

	cerrarArchivos(archivos)

	e.mu.Lock()
	proc.estado = "Terminado"
	proc.codigo = &codigo
	e.mu.Unlock()
}

func (e *ejecutor) estado_(p comun.Mensaje) comun.Mensaje {
	id := comun.Texto(p, "id-ejecucion")

	e.mu.Lock()
	defer e.mu.Unlock()

	if id == "" {
		lista := []comun.Mensaje{}

		for _, proc := range e.procesos {
			lista = append(lista, infoProc(proc))
		}

		return comun.Ok(comun.Mensaje{
			"procesos": lista,
		})
	}

	proc, ok := e.procesos[id]
	if !ok {
		return comun.Error("proceso no encontrado")
	}

	return comun.Ok(infoProc(proc))
}

func (e *ejecutor) matar(p comun.Mensaje) comun.Mensaje {
	id := comun.Texto(p, "id-ejecucion")

	if id == "" {
		return comun.Error("falta campo: id-ejecucion")
	}

	e.mu.Lock()
	proc, ok := e.procesos[id]
	e.mu.Unlock()

	if !ok {
		return comun.Error("proceso no encontrado")
	}

	if proc.estado == "Terminado" {
		return comun.Error("proceso no encontrado o ya terminado")
	}

	if err := proc.cmd.Process.Kill(); err != nil {
		return comun.Error("no se pudo matar el proceso")
	}

	return comun.Ok(nil)
}

func (e *ejecutor) suspender() comun.Mensaje {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, proc := range e.procesos {
		if proc.estado == "Ejecutando" {
			if err := suspenderPID(proc.cmd.Process.Pid); err != nil {
				return comun.Error("no se pudo suspender procesos")
			}

			proc.estado = "Suspendido"
		}
	}

	e.estado = "Suspendido"

	return comun.Ok(nil)
}

func (e *ejecutor) reasumir() comun.Mensaje {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, proc := range e.procesos {
		if proc.estado == "Suspendido" {
			if err := reanudarPID(proc.cmd.Process.Pid); err != nil {
				return comun.Error("no se pudo reanudar procesos")
			}

			proc.estado = "Ejecutando"
		}
	}

	e.estado = "Corriendo"

	return comun.Ok(nil)
}

func (e *ejecutor) parar() comun.Mensaje {
	e.mu.Lock()
	e.estado = "Parar"
	e.mu.Unlock()

	for {
		e.mu.Lock()

		activos := 0

		for _, proc := range e.procesos {
			if proc.estado == "Suspendido" {
				reanudarPID(proc.cmd.Process.Pid)
				proc.estado = "Ejecutando"
			}

			if proc.estado == "Ejecutando" {
				activos++
			}
		}

		e.mu.Unlock()

		if activos == 0 {
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	e.mu.Lock()
	e.estado = "Terminado"
	e.mu.Unlock()

	return comun.Ok(nil)
}

func infoProc(proc *proceso) comun.Mensaje {
	r := comun.Mensaje{
		"id-ejecucion":   proc.id,
		"id-programa":    proc.progID,
		"proceso-estado": proc.estado,
	}

	if proc.codigo != nil {
		r["codigo-salida"] = *proc.codigo
	}

	return r
}

func abrirFichero(aralmac string, id string, escritura bool) (*os.File, error) {
	if id == "" {
		return nil, nil
	}

	ruta := filepath.Join(aralmac, "ficheros", id+".txt")

	if escritura {
		return os.OpenFile(ruta, os.O_WRONLY|os.O_TRUNC, 0644)
	}

	return os.Open(ruta)
}

func cerrarArchivos(archivos []*os.File) {
	for _, archivo := range archivos {
		if archivo != nil {
			archivo.Close()
		}
	}
}
