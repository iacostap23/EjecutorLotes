# Diseño del sistema: Ejecutor de lotes en Go

## 1. Propósito del proyecto

Este proyecto implementa un sistema de ejecución de lotes usando el lenguaje Go. La idea principal es tener varios servicios separados que se comuniquen mediante tuberías nombradas y mensajes JSON.

El sistema permite que un cliente envíe solicitudes para administrar ficheros, registrar programas, ejecutar procesos, consultar estados, suspender/reanudar servicios y finalizar el sistema de forma ordenada.

El diseño busca funcionar tanto en Windows como en Linux/WSL, manteniendo el mismo protocolo lógico de comunicación aunque internamente cada sistema operativo maneje las tuberías de manera diferente.

---

## 2. Componentes principales

El sistema está dividido en cinco componentes principales:

| Componente | Responsabilidad principal |
|---|---|
| `cliente` | Envía mensajes JSON hacia `ctrllt` y muestra la respuesta recibida. |
| `ctrllt` | Recibe las solicitudes del cliente, identifica el servicio destino y redirige la petición. |
| `gesfich` | Administra ficheros dentro de `aralmac`. |
| `gesprog` | Registra programas ejecutables y los copia dentro de `aralmac`. |
| `ejecutor` | Ejecuta programas registrados, administra sus estados y controla procesos activos. |

Además, existe un paquete auxiliar:

| Paquete | Responsabilidad |
|---|---|
| `pipes` | Abstrae la comunicación por tuberías nombradas en Windows y Linux. |
| `comun` | Contiene funciones comunes para mensajes JSON, respuestas, listas, textos e identificadores. |

---

## 3. Arquitectura general

El flujo general del sistema es:

```text
cliente
   |
   | JSON por tubería nombrada
   v
ctrllt
   |
   | Redirige según el campo "servicio"
   |
   +----> gesfich
   |
   +----> gesprog
   |
   +----> ejecutor
```

El cliente no se comunica directamente con `gesfich`, `gesprog` ni `ejecutor`. El punto central de entrada es `ctrllt`.

Esto permite que el cliente use un único canal lógico y que `ctrllt` actúe como coordinador del sistema.

---

## 4. Comunicación por tuberías nombradas

La comunicación se realiza mediante tuberías nombradas.

Cada servicio tiene una tubería para recibir peticiones y otra para enviar respuestas. Esto permite manejar la comunicación de forma homogénea entre Windows y Linux/WSL.

Ejemplo de tuberías usadas en las pruebas:

| Servicio | Petición | Respuesta |
|---|---|---|
| Cliente a `ctrllt` | `lotes-ctrllt-req` | `lotes-ctrllt-res` |
| `ctrllt` a `gesfich` | `lotes-gesfich-req` | `lotes-gesfich-res` |
| `ctrllt` a `gesprog` | `lotes-gesprog-req` | `lotes-gesprog-res` |
| `ctrllt` a `ejecutor` | `lotes-ejecutor-req` | `lotes-ejecutor-res` |

En Linux/WSL, las tuberías se crean dentro de `/tmp` cuando se usa un nombre simple. Por ejemplo:

```text
/tmp/lotes-ctrllt-req
/tmp/lotes-ctrllt-res
```

En Windows se usa la implementación correspondiente de tuberías nombradas de Windows.

---

## 5. Diferencias entre Windows y Linux/WSL

El proyecto mantiene el mismo protocolo lógico en ambos sistemas operativos, pero la implementación interna de las tuberías cambia.

### Windows

En Windows se utiliza una implementación de tuberías nombradas compatible con el sistema operativo. El objetivo es que el cliente pueda enviar una petición y recibir una respuesta usando el mismo protocolo JSON.

### Linux/WSL

En Linux/WSL se usan FIFOs. Como las FIFOs son de comunicación simple, se manejan tuberías separadas para petición y respuesta.

Para soportar varios clientes al mismo tiempo, la implementación de Linux usa una estrategia de respuesta por petición. En términos simples, cada cliente puede recibir su propia respuesta sin mezclarse con la respuesta de otro cliente.

Esta decisión fue importante porque al probar varios clientes simultáneos se necesitaba evitar respuestas cruzadas o clientes bloqueados.

---

## 6. Protocolo JSON

Todos los mensajes principales se envían en formato JSON.

Una petición tiene esta forma general:

```json
{
  "servicio": "gesfich",
  "operacion": "Crear"
}
```

Una respuesta exitosa tiene esta forma general:

```json
{
  "estado": "ok"
}
```

Una respuesta con error tiene esta forma general:

```json
{
  "estado": "error",
  "mensaje": "descripcion del error"
}
```

El protocolo usa mensajes terminados en salto de línea para que puedan ser leídos como líneas completas desde las tuberías.

---

## 7. Identificadores usados

El sistema usa identificadores secuenciales para ficheros, programas y ejecuciones.

| Elemento | Formato | Ejemplo |
|---|---|---|
| Fichero | `f-XXXX` | `f-0001` |
| Programa | `p-XXXX` | `p-0001` |
| Ejecución | `e-XXXX` | `e-0001` |

Estos identificadores permiten referenciar los recursos después de crearlos o registrarlos.

---

## 8. Almacenamiento en `aralmac`

El directorio `aralmac` funciona como almacenamiento local del sistema.

Dentro de `aralmac` se guardan los ficheros, los programas registrados, los metadatos y los contadores necesarios para generar identificadores.

Estructura general esperada:

```text
aralmac/
  ficheros/
    f-0001.txt
    f-0002.txt
  programas/
    p-0001.json
    p-0001/
      procesar.exe
  contador-ficheros.txt
  contador-programas.txt
```

En Linux el ejecutable no tiene extensión `.exe`, por ejemplo:

```text
aralmac/programas/p-0001/procesar
```

En Windows puede quedar así:

```text
aralmac/programas/p-0001/procesar.exe
```

---

## 9. Servicio `gesfich`

`gesfich` administra los ficheros almacenados dentro de `aralmac`.

### Operaciones soportadas

| Operación | Descripción |
|---|---|
| `Crear` | Crea un nuevo fichero vacío y retorna un `id-fichero`. |
| `Leer` | Lee un fichero por ID o lista todos los ficheros si no se envía ID. |
| `Actualizar` | Copia el contenido de una ruta externa hacia el fichero registrado. |
| `Borrar` | Elimina un fichero por ID. |
| `Suspender` | Suspende operaciones de escritura o modificación. |
| `Reasumir` | Reactiva el servicio. |
| `Terminar` | Finaliza el servicio. |

### Ejemplo: crear fichero

Petición:

```json
{
  "servicio": "gesfich",
  "operacion": "Crear"
}
```

Respuesta:

```json
{
  "estado": "ok",
  "id-fichero": "f-0001"
}
```

### Ejemplo: leer fichero

Petición:

```json
{
  "servicio": "gesfich",
  "operacion": "Leer",
  "id-fichero": "f-0001"
}
```

Respuesta:

```json
{
  "estado": "ok",
  "contenido": "hola mundo\n"
}
```

---

## 10. Servicio `gesprog`

`gesprog` administra los programas ejecutables registrados en el sistema.

Una decisión importante del diseño es que `gesprog` no solo guarda la ruta original del ejecutable, sino que copia el ejecutable dentro de `aralmac`.

Esto se hizo para cumplir mejor con la práctica, porque el programa queda bajo control del almacenamiento del sistema.

### Operaciones soportadas

| Operación | Descripción |
|---|---|
| `Guardar` | Recibe un ejecutable, lo copia dentro de `aralmac` y retorna un `id-programa`. |
| `Leer` | Lee un programa por ID o lista todos los programas registrados. |
| `Actualizar` | Reemplaza el ejecutable asociado a un programa. |
| `Borrar` | Elimina el registro y la copia del programa. |
| `Suspender` | Suspende operaciones de escritura o modificación. |
| `Reasumir` | Reactiva el servicio. |
| `Terminar` | Finaliza el servicio. |

### Almacenamiento de programas

Cuando se registra un programa, se crea una estructura como esta:

```text
aralmac/programas/p-0001.json
aralmac/programas/p-0001/procesar.exe
```

El archivo JSON guarda los metadatos:

```json
{
  "id-programa": "p-0001",
  "nombre": "procesar.exe",
  "ejecutable": "aralmac/programas/p-0001/procesar.exe",
  "args": [],
  "env": []
}
```

En Linux el ejecutable puede llamarse simplemente `procesar`.

### Ejemplo: guardar programa

Petición:

```json
{
  "servicio": "gesprog",
  "operacion": "Guardar",
  "ejecutable": "./programas/procesar.exe",
  "args": [],
  "env": []
}
```

Respuesta:

```json
{
  "estado": "ok",
  "id-programa": "p-0001"
}
```

---

## 11. Servicio `ejecutor`

`ejecutor` se encarga de ejecutar los programas registrados por `gesprog`.

El ejecutor consulta los metadatos del programa, obtiene la ruta del ejecutable copiado dentro de `aralmac` y lanza el proceso.

### Operaciones soportadas

| Operación | Descripción |
|---|---|
| `Ejecutar` | Ejecuta un programa registrado y retorna un `id-ejecucion`. |
| `Estado` | Consulta el estado de una ejecución o de todas las ejecuciones. |
| `Matar` | Termina una ejecución activa. |
| `Suspender` | Suspende el servicio y los procesos activos. |
| `Reasumir` | Reanuda el servicio y los procesos suspendidos. |
| `Parar` / `Terminar` | Finaliza el servicio de forma ordenada. |

### Estados de ejecución

| Estado | Significado |
|---|---|
| `Ejecutando` | El proceso sigue activo. |
| `Suspendido` | El proceso fue suspendido temporalmente. |
| `Terminado` | El proceso finalizó o fue terminado. |

### Entrada, salida y error

La operación `Ejecutar` puede recibir ficheros para:

| Campo | Uso |
|---|---|
| `stdin` | Fichero usado como entrada estándar. |
| `stdout` | Fichero donde se guarda la salida estándar. |
| `stderr` | Fichero donde se guarda la salida de error. |

Ejemplo:

```json
{
  "servicio": "ejecutor",
  "operacion": "Ejecutar",
  "id-programa": "p-0001",
  "stdin": "f-0001",
  "stdout": "f-0002",
  "stderr": "f-0003"
}
```

Respuesta:

```json
{
  "estado": "ok",
  "id-ejecucion": "e-0001"
}
```

---

## 12. Servicio `ctrllt`

`ctrllt` es el servicio coordinador.

Su responsabilidad es recibir las peticiones del cliente, revisar el campo `servicio` y reenviar el mensaje al servicio correspondiente.

### Servicios reconocidos

| Valor de `servicio` | Destino |
|---|---|
| `gesfich` | Servicio de ficheros. |
| `gesprog` | Servicio de programas. |
| `ejecutor` | Servicio de ejecución. |
| `ctrllt` | Operaciones propias del controlador. |

### Operación `Terminar`

Cuando `ctrllt` recibe:

```json
{
  "servicio": "ctrllt",
  "operacion": "Terminar"
}
```

debe finalizar el sistema de forma ordenada.

El diseño aplicado es:

1. Enviar orden de cierre a `gesfich`.
2. Enviar orden de cierre a `gesprog`.
3. Enviar orden de parada a `ejecutor`.
4. Esperar el cierre del ejecutor si hay procesos activos.
5. Responder `{"estado":"ok"}`.
6. Finalizar el propio `ctrllt`.

También se validó el caso en el que existe un proceso activo al momento de ejecutar `Terminar`.

---

## 13. Suspensión y reanudación

Los servicios implementan operaciones de suspensión y reanudación.

### En `gesfich` y `gesprog`

Cuando el servicio está suspendido:

- Se permite `Leer`.
- Se permite `Reasumir`.
- Se permite `Terminar`.
- Se bloquean operaciones que modifican el estado, como `Crear`, `Guardar`, `Actualizar` o `Borrar`.

### En `ejecutor`

Cuando el ejecutor está suspendido:

- No acepta nuevas ejecuciones.
- Los procesos activos pueden quedar en estado `Suspendido`.
- Al ejecutar `Reasumir`, los procesos suspendidos continúan.
- Se comprobó que un proceso reanudado puede terminar naturalmente.

---

## 14. Manejo de errores

El sistema devuelve errores en formato JSON cuando una operación no se puede realizar.

Ejemplos de errores controlados:

| Caso | Respuesta esperada |
|---|---|
| Fichero inexistente | `{"estado":"error","mensaje":"fichero no encontrado"}` |
| Programa inexistente | `{"estado":"error","mensaje":"programa no encontrado"}` |
| Ejecución inexistente | `{"estado":"error","mensaje":"proceso no encontrado"}` |
| Servicio desconocido | `{"estado":"error","mensaje":"servicio desconocido"}` |
| Operación desconocida | `{"estado":"error","mensaje":"operacion desconocida"}` |
| Servicio suspendido | `{"estado":"error","mensaje":"servicio suspendido"}` |

---

## 15. Scripts de limpieza

| Script | Sistema | Propósito |
|---|---|---|
| `scripts/limpiar_windows.ps1` | Windows | Borra archivos generados, binarios, logs y `aralmac`. |
| `scripts/limpiar_linux.sh` | Linux/WSL | Borra archivos generados, binarios, logs, `aralmac` y tuberías temporales. |


### Windows

Para limpiar archivos generados:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\scripts\limpiar_windows.ps1
```

### Linux/WSL

Para limpiar archivos generados:

```bash
chmod +x scripts/limpiar_linux.sh
./scripts/limpiar_linux.sh
```
---

## 16. Conclusión

El diseño implementado cumple con el objetivo de construir un sistema de ejecución de lotes basado en servicios, tuberías nombradas y mensajes JSON.

El sistema funciona en Windows y Linux/WSL, permite administrar ficheros y programas, ejecutar procesos, consultar estados, suspender y reanudar servicios, manejar errores, ejecutar múltiples procesos, atender múltiples clientes y finalizar de manera ordenada.

