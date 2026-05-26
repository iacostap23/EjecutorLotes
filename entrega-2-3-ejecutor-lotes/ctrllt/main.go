package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"ejecutor-lotes-go/comun"
	"ejecutor-lotes-go/pipes"
)

// canal agrupa las dos tuberías de un servicio (petición y respuesta)
type canal struct {
	req string
	res string
}

func main() {
	cReq := flag.String("c", "lotes-ctrllt-req", "tuberia peticiones cliente")
	cRes := flag.String("a", "lotes-ctrllt-res", "tuberia respuestas cliente")
	fReq := flag.String("f", "lotes-gesfich-req", "tuberia peticiones gesfich")
	fRes := flag.String("b", "lotes-gesfich-res", "tuberia respuestas gesfich")
	pReq := flag.String("p", "lotes-gesprog-req", "tuberia peticiones gesprog")
	pRes := flag.String("s", "lotes-gesprog-res", "tuberia respuestas gesprog")
	eReq := flag.String("e", "lotes-ejecutor-req", "tuberia peticiones ejecutor")
	eRes := flag.String("d", "lotes-ejecutor-res", "tuberia respuestas ejecutor")
	flag.Parse()

	servicios := map[string]canal{
		"gesfich":  {*fReq, *fRes},
		"gesprog":  {*pReq, *pRes},
		"ejecutor": {*eReq, *eRes},
	}

	err := pipes.Servidor(*cReq, *cRes, func(linea string) (string, bool) {
		var p comun.Mensaje
		if err := json.Unmarshal([]byte(linea), &p); err != nil {
			return comun.AJson(comun.Error("json invalido")), false
		}

		servicio := comun.Texto(p, "servicio")
		operacion := comun.Texto(p, "operacion")

		if servicio == "ctrllt" && operacion == "Terminar" {
			apagarSistema(servicios)
			return comun.AJson(comun.Ok(nil)), true
		}

		dest, ok := servicios[servicio]
		if !ok {
			return comun.AJson(comun.Error("servicio desconocido")), false
		}
		respuesta, err := pipes.Enviar(dest.req, dest.res, linea)
		if err != nil {
			return comun.AJson(comun.Error("servicio no conectado")), false
		}
		return respuesta, false
	})

	if err != nil {
		fmt.Println(comun.AJson(comun.Error("error en tuberia ctrllt")))
	}
}

// apagarSistema termina gesfich y gesprog, y ordena parar al ejecutor
func apagarSistema(s map[string]canal) {
	pipes.Enviar(s["gesfich"].req, s["gesfich"].res, `{"servicio":"gesfich","operacion":"Terminar"}`)
	pipes.Enviar(s["gesprog"].req, s["gesprog"].res, `{"servicio":"gesprog","operacion":"Terminar"}`)
	pipes.Enviar(s["ejecutor"].req, s["ejecutor"].res, `{"servicio":"ejecutor","operacion":"Parar"}`)
}
