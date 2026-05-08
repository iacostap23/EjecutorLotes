# Diseño de API - Ejecutor de Lotes

**Curso:** Sistemas Operativos
**Práctica:** Ejecutor de Lotes
**Documento:** `docs/Diseño.md`

---

## Tabla de contenidos

1. [Introducción](#1-introducción)
2. [Alcance y criterios de diseño](#2-alcance-y-criterios-de-diseño)
3. [Componentes del sistema](#3-componentes-del-sistema)
4. [Modelo de comunicación](#4-modelo-de-comunicación)
5. [Formato estándar de mensajes JSON](#5-formato-estándar-de-mensajes-json)
6. [Identificadores del sistema](#6-identificadores-del-sistema)
7. [Modos de comunicación y enrutamiento](#7-modos-de-comunicación-y-enrutamiento)
8. [API de `gesprog`](#8-api-de-gesprog)
9. [API de `gesfich`](#9-api-de-gesfich)
10. [API de `ejecutor`](#10-api-de-ejecutor)
11. [API de `ctrllt`](#11-api-de-ctrllt)
12. [Resumen general de operaciones](#12-resumen-general-de-operaciones)
13. [Catálogo general de errores](#13-catálogo-general-de-errores)
14. [Consideraciones sobre `aralmac`](#14-consideraciones-sobre-aralmac)
15. [Orden de arranque recomendado](#15-orden-de-arranque-recomendado)
16. [Máquinas de estado](#16-máquinas-de-estado)
17. [Ejemplo completo de flujo](#17-ejemplo-completo-de-flujo)
18. [Conclusión](#18-conclusión)

---

## 1. Introducción

Este documento presenta el diseño de la API para la práctica **Ejecutor de Lotes**. El sistema simula un mecanismo de ejecución por lotes en el que primero se registran programas y ficheros, y posteriormente se ejecutan procesos usando los identificadores generados durante dichos registros.

La API se define como un **contrato de comunicación entre procesos**, donde los componentes intercambian mensajes en formato **JSON** mediante **tuberías nombradas**.

El diseño establece:

* La estructura estándar de las peticiones JSON.
* La estructura estándar de las respuestas JSON.
* Los servicios disponibles en la API.
* Las operaciones soportadas por cada servicio.
* Los datos requeridos para cada operación.
* Los errores posibles por operación.
* Las máquinas de estado de los servicios.
* La relación entre `cliente`, `ctrllt`, `gesprog`, `gesfich`, `ejecutor` y `aralmac`.

---

## 2. Alcance y criterios de diseño

Los criterios adoptados son:

1. El formato de los mensajes será JSON.
2. Cada petición tendrá un identificador único `id_peticion`.
3. Cada respuesta devolverá el mismo `id_peticion` recibido.
4. El campo `servicio` indicará el servicio destino de la petición.
5. La API podrá usarse de dos formas:

   * Cliente conectado directamente a un servicio (`gesprog`, `gesfich` o `ejecutor`).
   * Cliente conectado a `ctrllt`, que actúa como pasarela hacia los servicios internos.
6. Para esta versión se considera un único cliente de prueba.
7. En Linux se usarán tuberías nombradas half-duplex.
8. En Windows se usarán tuberías nombradas full-duplex.
9. Los ficheros se manejarán como texto plano codificado en UTF-8.
10. El ejecutor podrá tener varios lotes activos simultáneamente.
11. La operación `ejecutar` responderá inmediatamente con un `id_lote`; la ejecución del lote continuará en segundo plano.
12. El ejecutor accederá directamente a `aralmac` para consultar programas y ficheros registrados.
13. Las tuberías se consideran parte de la capa de transporte y no se incluyen dentro del cuerpo JSON.

---

## 3. Componentes del sistema

| Componente | Función                                                                                                                                                                                   |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cliente`  | Proceso que genera peticiones JSON. Puede conectarse directamente a `gesprog`, `gesfich`, `ejecutor` o a `ctrllt`.                                                                        |
| `ctrllt`   | Servicio de control de lotes. Recibe peticiones, analiza el campo `servicio`, reenvía la petición al servicio correspondiente y devuelve la respuesta al cliente.                         |
| `gesprog`  | Servicio de gestión de programas. Guarda, lee, actualiza, borra y controla programas almacenados en `aralmac`.                                                                            |
| `gesfich`  | Servicio de gestión de ficheros. Crea, lee, actualiza, borra y controla ficheros almacenados en `aralmac`.                                                                                |
| `ejecutor` | Servicio de ejecución de lotes. Ejecuta procesos de lote usando programas y ficheros registrados en `aralmac`; también permite consultar, matar, suspender, reasumir y parar ejecuciones. |
| `aralmac`  | Área de almacenamiento donde se guardan programas, ficheros y metadatos. No es un servicio activo, sino una región de almacenamiento compartida por los servicios.                        |

Arquitectura general del sistema:

```text
cliente
   |
   |--------------------------→ gesprog
   |--------------------------→ gesfich
   |--------------------------→ ejecutor
   |
   v
ctrllt
   |--------------------------→ gesprog
   |--------------------------→ gesfich
   |--------------------------→ ejecutor

               gesprog ───┐
               gesfich ───┼──→ aralmac
               ejecutor ──┘
```

`ctrllt` funciona como pasarela, pero el diseño también permite que el cliente se conecte directamente con los servicios.

---

## 4. Modelo de comunicación

### 4.1. Tuberías nombradas

La comunicación entre procesos se realiza mediante tuberías nombradas. Cada tubería debe tener un nombre único.

El modelo de tuberías depende del sistema operativo:

| Sistema operativo | Tipo de tubería | Estructura de comunicación                                                   |
| ----------------- | --------------- | ---------------------------------------------------------------------------- |
| Linux             | Half-duplex     | Se usan dos tuberías por enlace: una para peticiones y otra para respuestas. |
| Windows           | Full-duplex     | Se puede usar una sola tubería por enlace para enviar y recibir.             |

---

### 4.2. Responsabilidad de creación de tuberías

Para evitar ambigüedad en la sinopsis de `ctrllt`, este diseño usa una opción diferente para cada tubería. En particular, la tubería de respuesta de `gesprog` se identifica con `-s`, en lugar de reutilizar `-c`, porque `-c` ya se emplea para la tubería de petición del cliente. La opción `-s` representa la salida o respuesta de `gesprog`.

Sinopsis propuesta para `ctrllt`:

```bash
ctrllt -c <pipe-cliente-req> [-a <pipe-cliente-res>] \
       -f <pipe-gesfich-req> [-b <pipe-gesfich-res>] \
       -p <pipe-gesprog-req> [-s <pipe-gesprog-res>] \
       -e <pipe-ejecutor-req> [-d <pipe-ejecutor-res>]
```

Cada servicio será responsable de crear las tuberías nombradas por las que recibe peticiones. En sistemas half-duplex, también deberá crear la tubería correspondiente para enviar respuestas.

| Componente           | Responsabilidad                                                                                           |
| -------------------- | --------------------------------------------------------------------------------------------------------- |
| `ctrllt`             | Crea las tuberías para recibir peticiones del cliente y enviar respuestas al cliente.                     |
| `gesprog`            | Crea sus tuberías de comunicación para recibir peticiones y enviar respuestas relacionadas con programas. |
| `gesfich`            | Crea sus tuberías de comunicación para recibir peticiones y enviar respuestas relacionadas con ficheros.  |
| `ejecutor`           | Crea sus tuberías de comunicación para recibir peticiones y enviar respuestas relacionadas con lotes.     |
| `cliente`            | No crea las tuberías principales; se conecta a las tuberías creadas por el servicio destino.              |
| `ctrllt` como emisor | Cuando reenvía una petición, se conecta a las tuberías ya creadas por `gesprog`, `gesfich` o `ejecutor`.  |

Esta decisión evita que dos procesos intenten crear la misma tubería y mantiene una responsabilidad clara: el proceso que ofrece el servicio crea el canal por el que recibirá peticiones.

---

### 4.3. Tuberías en Linux

En Linux, por tratarse de tuberías half-duplex, cada enlace usa dos tuberías.

| Comunicación          | Petición                  | Respuesta                 |
| --------------------- | ------------------------- | ------------------------- |
| Cliente ↔ `ctrllt`    | `/tmp/lotes-ctrllt-req`   | `/tmp/lotes-ctrllt-res`   |
| Cliente ↔ `gesprog`   | `/tmp/lotes-gesprog-req`  | `/tmp/lotes-gesprog-res`  |
| Cliente ↔ `gesfich`   | `/tmp/lotes-gesfich-req`  | `/tmp/lotes-gesfich-res`  |
| Cliente ↔ `ejecutor`  | `/tmp/lotes-ejecutor-req` | `/tmp/lotes-ejecutor-res` |
| `ctrllt` ↔ `gesprog`  | `/tmp/lotes-gesprog-req`  | `/tmp/lotes-gesprog-res`  |
| `ctrllt` ↔ `gesfich`  | `/tmp/lotes-gesfich-req`  | `/tmp/lotes-gesfich-res`  |
| `ctrllt` ↔ `ejecutor` | `/tmp/lotes-ejecutor-req` | `/tmp/lotes-ejecutor-res` |

Los modos de comunicación directa y vía `ctrllt` se consideran alternativas de uso. Para esta primera versión no se asume que el cliente directo y `ctrllt` escriban simultáneamente sobre la misma tubería de un servicio.

---

### 4.4. Tuberías en Windows

En Windows, al trabajar con tuberías full-duplex, puede usarse una sola tubería por enlace.

| Comunicación          | Tubería full-duplex       |
| --------------------- | ------------------------- |
| Cliente ↔ `ctrllt`    | `\\.\pipe\lotes-ctrllt`   |
| Cliente ↔ `gesprog`   | `\\.\pipe\lotes-gesprog`  |
| Cliente ↔ `gesfich`   | `\\.\pipe\lotes-gesfich`  |
| Cliente ↔ `ejecutor`  | `\\.\pipe\lotes-ejecutor` |
| `ctrllt` ↔ `gesprog`  | `\\.\pipe\lotes-gesprog`  |
| `ctrllt` ↔ `gesfich`  | `\\.\pipe\lotes-gesfich`  |
| `ctrllt` ↔ `ejecutor` | `\\.\pipe\lotes-ejecutor` |

---

### 4.5. Relación entre tuberías y JSON

Las tuberías pertenecen a la capa de transporte. El JSON representa el contenido lógico del mensaje. Por esta razón, los nombres de tuberías no hacen parte del cuerpo JSON.

La identificación de una petición se realiza mediante `id_peticion`. La respuesta debe conservar el mismo `id_peticion` para que el cliente pueda relacionarla con la petición original.

---

### 4.6. Delimitación de mensajes

Cada mensaje JSON se enviará codificado en UTF-8 y terminado con un salto de línea:

```text
\n
```

El receptor leerá hasta encontrar dicho delimitador para reconstruir cada mensaje completo.

---

## 5. Formato estándar de mensajes JSON

Todos los componentes usarán una estructura estándar para peticiones, respuestas exitosas y respuestas con error.

---

### 5.1. Formato estándar de petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0001",
  "servicio": "gesprog",
  "operacion": "guardar",
  "datos": {}
}
```

| Campo          | Tipo   | Obligatorio | Descripción                                                                   |
| -------------- | ------ | ----------: | ----------------------------------------------------------------------------- |
| `tipo_mensaje` | cadena |          Sí | Siempre será `peticion`.                                                      |
| `id_cliente`   | cadena |          Sí | Identificador lógico del cliente.                                             |
| `id_peticion`  | cadena |          Sí | Identificador único de la petición.                                           |
| `servicio`     | cadena |          Sí | Servicio destino de la petición: `gesprog`, `gesfich`, `ejecutor` o `ctrllt`. |
| `operacion`    | cadena |          Sí | Operación solicitada al servicio.                                             |
| `datos`        | objeto |          Sí | Parámetros de la operación. Si no hay parámetros, se usa `{}`.                |

Aunque esta versión contempla un único cliente de prueba, se conserva `id_cliente` para mantener trazabilidad en los mensajes.

---

### 5.2. Formato estándar de respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0001",
  "estado": "ok",
  "resultado": {},
  "error": null,
  "mensaje": "Operación realizada correctamente"
}
```

| Campo          | Tipo   | Descripción                                    |
| -------------- | ------ | ---------------------------------------------- |
| `tipo_mensaje` | cadena | Siempre será `respuesta`.                      |
| `id_cliente`   | cadena | Mismo valor recibido en la petición.           |
| `id_peticion`  | cadena | Mismo valor recibido en la petición.           |
| `estado`       | cadena | `ok` cuando la operación fue exitosa.          |
| `resultado`    | objeto | Información producida por la operación.        |
| `error`        | nulo   | En una respuesta exitosa será `null`.          |
| `mensaje`      | cadena | Mensaje descriptivo de la operación realizada. |

---

### 5.3. Formato estándar de respuesta con error

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0001",
  "estado": "error",
  "resultado": {},
  "error": {
    "codigo": "PROGRAMA_NO_EXISTE",
    "detalle": "No existe un programa registrado con id p-9999"
  },
  "mensaje": "No fue posible completar la operación"
}
```

| Campo           | Tipo   | Descripción                                       |
| --------------- | ------ | ------------------------------------------------- |
| `tipo_mensaje`  | cadena | Siempre será `respuesta`.                         |
| `id_cliente`    | cadena | Mismo valor recibido en la petición.              |
| `id_peticion`   | cadena | Mismo valor recibido en la petición.              |
| `estado`        | cadena | `error` cuando la operación no pudo completarse.  |
| `resultado`     | objeto | En respuestas con error se usa `{}`.              |
| `error.codigo`  | cadena | Código del error ocurrido.                        |
| `error.detalle` | cadena | Descripción específica del error.                 |
| `mensaje`       | cadena | Mensaje general indicando que la operación falló. |

Regla general:

```text
Si estado = ok:
    resultado contiene los datos útiles.
    error = null.

Si estado = error:
    resultado = {}.
    error contiene codigo y detalle.
```

---

## 6. Identificadores del sistema

| Identificador | Formato    | Ejemplo    | Lo genera  | Descripción                                    |
| ------------- | ---------- | ---------- | ---------- | ---------------------------------------------- |
| `id_cliente`  | `cli-XXXX` | `cli-0001` | Cliente    | Identifica al cliente que realiza la petición. |
| `id_peticion` | `req-XXXX` | `req-0001` | Cliente    | Identifica una petición específica.            |
| `id_programa` | `p-XXXX`   | `p-0001`   | `gesprog`  | Identifica un programa registrado.             |
| `id_fichero`  | `f-XXXX`   | `f-0001`   | `gesfich`  | Identifica un fichero registrado.              |
| `id_lote`     | `l-XXXX`   | `l-0001`   | `ejecutor` | Identifica una ejecución de lote. Su contador se reinicia en cada arranque del ejecutor. |

*Nota:* Los identificadores `id_programa` e `id_fichero` deben conservar unicidad mientras existan los registros asociados en `aralmac`. El `id_lote` es efímero: la numeración comienza desde `l-0001` cada vez que el ejecutor se inicia.

---

## 7. Modos de comunicación y enrutamiento

### 7.1. Modo directo

En modo directo, el cliente se conecta directamente al servicio que debe atender la petición.

| Campo `servicio` | Servicio que atiende          |
| ---------------- | ----------------------------- |
| `gesprog`        | Gestión de programas.         |
| `gesfich`        | Gestión de ficheros.          |
| `ejecutor`       | Gestión y ejecución de lotes. |

Ejemplo:

```text
cliente → gesprog
cliente → gesfich
cliente → ejecutor
```

---

### 7.2. Modo con `ctrllt`

En modo pasarela, el cliente se conecta a `ctrllt`. El control recibe la petición, revisa el campo `servicio` y reenvía el mensaje al servicio correspondiente.

| Campo `servicio` recibido por `ctrllt` | Servicio destino             |
| -------------------------------------- | ---------------------------- |
| `gesprog`                              | `gesprog`                    |
| `gesfich`                              | `gesfich`                    |
| `ejecutor`                             | `ejecutor`                   |
| `ctrllt`                               | Operación propia de `ctrllt` |

Ejemplo:

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0001",
  "servicio": "gesprog",
  "operacion": "guardar",
  "datos": {
    "ejecutable": "/home/usuario/programas/actividad.py",
    "argumentos": ["sumar"],
    "ambiente": {}
  }
}
```

Como `servicio` es `gesprog`, `ctrllt` reenvía la petición a `gesprog`.

`ctrllt` no guarda programas, no crea ficheros y no ejecuta lotes. Su función es recibir, clasificar, reenviar y devolver la respuesta al cliente.

---

## 8. API de `gesprog`

`gesprog` administra los programas registrados en `aralmac`. Un programa registrado contiene:

* `id_programa`
* ejecutable
* argumentos
* ambiente
* descripción opcional

En este diseño, las operaciones de `gesprog` usan `id_programa`, debido a que este servicio administra programas.

### Estados y operaciones permitidas

| Estado de `gesprog` | Operaciones permitidas                                             |
| ------------------- | ------------------------------------------------------------------ |
| `corriendo`         | `guardar`, `leer`, `actualizar`, `borrar`, `suspender`, `terminar` |
| `suspendido`        | `leer`, `reasumir`, `terminar`                                     |
| `terminado`         | Ninguna operación normal.                                          |

---

### 8.1. Operación `guardar`

Registra un nuevo programa ejecutable en `aralmac`.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0001",
  "servicio": "gesprog",
  "operacion": "guardar",
  "datos": {
    "ejecutable": "/home/usuario/programas/actividad.py",
    "argumentos": ["sumar"],
    "ambiente": {
      "PYTHONUNBUFFERED": "1"
    },
    "descripcion": "Programa que suma números recibidos por entrada estándar"
  }
}
```

#### Datos requeridos

| Campo         | Tipo    | Obligatorio | Descripción                                                       |
| ------------- | ------- | ----------: | ----------------------------------------------------------------- |
| `ejecutable`  | cadena  |          Sí | Ruta o nombre del ejecutable válido. Puede ser binario o script.  |
| `argumentos`  | arreglo |          Sí | Argumentos que se usarán al ejecutar el programa. Puede ser `[]`. |
| `ambiente`    | objeto  |          Sí | Variables de ambiente. Puede ser `{}`.                            |
| `descripcion` | cadena  |          No | Descripción del programa.                                         |

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0001",
  "estado": "ok",
  "resultado": {
    "id_programa": "p-0001"
  },
  "error": null,
  "mensaje": "Programa registrado correctamente"
}
```

#### Errores posibles

| Código                       | Cuándo ocurre                                               |
| ---------------------------- | ----------------------------------------------------------- |
| `CAMPO_OBLIGATORIO_FALTANTE` | Falta `ejecutable`, `argumentos` o `ambiente`.              |
| `EJECUTABLE_NO_EXISTE`       | La ruta del ejecutable no existe.                           |
| `EJECUTABLE_INVALIDO`        | El archivo no puede ejecutarse o no es válido.              |
| `SERVICIO_SUSPENDIDO`        | `gesprog` está suspendido y la operación no está permitida. |
| `ERROR_ALMACENAMIENTO`       | No se pudo copiar o registrar el programa en `aralmac`.     |

---

### 8.2. Operación `leer`

Consulta un programa específico o lista todos los programas registrados.

#### Leer programa específico

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0002",
  "servicio": "gesprog",
  "operacion": "leer",
  "datos": {
    "id_programa": "p-0001"
  }
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0002",
  "estado": "ok",
  "resultado": {
    "id_programa": "p-0001",
    "ejecutable": "actividad.py",
    "argumentos": ["sumar"],
    "ambiente": {
      "PYTHONUNBUFFERED": "1"
    },
    "descripcion": "Programa que suma números recibidos por entrada estándar"
  },
  "error": null,
  "mensaje": "Programa encontrado"
}
```

#### Leer todos los programas

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0003",
  "servicio": "gesprog",
  "operacion": "leer",
  "datos": {}
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0003",
  "estado": "ok",
  "resultado": {
    "programas": [
      {
        "id_programa": "p-0001",
        "ejecutable": "actividad.py",
        "argumentos": ["sumar"],
        "descripcion": "Programa que suma números"
      },
      {
        "id_programa": "p-0002",
        "ejecutable": "dividir.py",
        "argumentos": ["2"],
        "descripcion": "Programa que divide entre 2"
      }
    ]
  },
  "error": null,
  "mensaje": "Listado de programas registrados"
}
```

#### Errores posibles

| Código                 | Cuándo ocurre                                   |
| ---------------------- | ----------------------------------------------- |
| `PROGRAMA_NO_EXISTE`   | Se envía un `id_programa` que no existe.        |
| `ERROR_ALMACENAMIENTO` | No se pudo acceder a la información almacenada. |

---

### 8.3. Operación `actualizar`

Actualiza la información de un programa registrado.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0004",
  "servicio": "gesprog",
  "operacion": "actualizar",
  "datos": {
    "id_programa": "p-0001",
    "ejecutable": "/home/usuario/programas/actividad_v2.py",
    "argumentos": ["promediar"],
    "ambiente": {},
    "descripcion": "Nueva versión del programa"
  }
}
```

#### Datos requeridos

| Campo         | Tipo    | Obligatorio | Descripción                                 |
| ------------- | ------- | ----------: | ------------------------------------------- |
| `id_programa` | cadena  |          Sí | Programa que se desea actualizar.           |
| `ejecutable`  | cadena  |          Sí | Nuevo ejecutable o nueva ruta del programa. |
| `argumentos`  | arreglo |          Sí | Nueva lista de argumentos.                  |
| `ambiente`    | objeto  |          Sí | Nuevas variables de ambiente.               |
| `descripcion` | cadena  |          No | Nueva descripción del programa.             |

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0004",
  "estado": "ok",
  "resultado": {
    "id_programa": "p-0001"
  },
  "error": null,
  "mensaje": "Programa actualizado correctamente"
}
```

#### Errores posibles

| Código                       | Cuándo ocurre                                               |
| ---------------------------- | ----------------------------------------------------------- |
| `PROGRAMA_NO_EXISTE`         | No existe el `id_programa`.                                 |
| `EJECUTABLE_NO_EXISTE`       | El nuevo ejecutable no existe.                              |
| `EJECUTABLE_INVALIDO`        | El nuevo ejecutable no es válido.                           |
| `CAMPO_OBLIGATORIO_FALTANTE` | Falta un campo requerido.                                   |
| `SERVICIO_SUSPENDIDO`        | `gesprog` está suspendido y la operación no está permitida. |
| `ERROR_ALMACENAMIENTO`       | No se pudo actualizar en `aralmac`.                         |

---

### 8.4. Operación `borrar`

Elimina un programa registrado.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0005",
  "servicio": "gesprog",
  "operacion": "borrar",
  "datos": {
    "id_programa": "p-0001"
  }
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0005",
  "estado": "ok",
  "resultado": {
    "id_programa": "p-0001"
  },
  "error": null,
  "mensaje": "Programa borrado correctamente"
}
```

#### Errores posibles

| Código                 | Cuándo ocurre                                               |
| ---------------------- | ----------------------------------------------------------- |
| `PROGRAMA_NO_EXISTE`   | No existe el `id_programa`.                                 |
| `PROGRAMA_EN_USO`      | El programa está siendo usado por un lote activo.           |
| `SERVICIO_SUSPENDIDO`  | `gesprog` está suspendido y la operación no está permitida. |
| `ERROR_ALMACENAMIENTO` | No se pudo borrar el programa.                              |

---

### 8.5. Operaciones `suspender`, `reasumir` y `terminar`

Controlan el estado del servicio `gesprog`.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0006",
  "servicio": "gesprog",
  "operacion": "suspender",
  "datos": {}
}
```

Para `reasumir` y `terminar` se usa la misma estructura, cambiando el valor de `operacion`.

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0006",
  "estado": "ok",
  "resultado": {
    "estado_servicio": "suspendido"
  },
  "error": null,
  "mensaje": "Servicio gesprog suspendido correctamente"
}
```

#### Errores posibles

| Código               | Cuándo ocurre                                      |
| -------------------- | -------------------------------------------------- |
| `OPERACION_INVALIDA` | La transición no es válida desde el estado actual. |
| `SERVICIO_TERMINADO` | El servicio ya está terminado.                     |

---

## 9. API de `gesfich`

`gesfich` administra los ficheros almacenados en `aralmac`. Un fichero puede ser fuente de entrada o destino de salida para procesos de lote.

Los ficheros se manejarán como texto plano codificado en UTF-8. No se usará Base64 para esta versión.

### Estados y operaciones permitidas

| Estado de `gesfich` | Operaciones permitidas                                           |
| ------------------- | ---------------------------------------------------------------- |
| `corriendo`         | `crear`, `leer`, `actualizar`, `borrar`, `suspender`, `terminar` |
| `suspendido`        | `leer`, `reasumir`, `terminar`                                   |
| `terminado`         | Ninguna operación normal.                                        |

---

### 9.1. Operación `crear`

Crea un fichero vacío en `aralmac` y devuelve un `id_fichero`.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0007",
  "servicio": "gesfich",
  "operacion": "crear",
  "datos": {
    "nombre_logico": "numeros_entrada"
  }
}
```

#### Datos requeridos

| Campo           | Tipo   | Obligatorio | Descripción                     |
| --------------- | ------ | ----------: | ------------------------------- |
| `nombre_logico` | cadena |          No | Nombre descriptivo del fichero. |

El fichero queda vacío hasta que se cargue contenido mediante `actualizar`.

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0007",
  "estado": "ok",
  "resultado": {
    "id_fichero": "f-0001"
  },
  "error": null,
  "mensaje": "Fichero creado correctamente"
}
```

#### Errores posibles

| Código                 | Cuándo ocurre                                               |
| ---------------------- | ----------------------------------------------------------- |
| `SERVICIO_SUSPENDIDO`  | `gesfich` está suspendido y la operación no está permitida. |
| `ERROR_ALMACENAMIENTO` | No se pudo crear el fichero en `aralmac`.                   |

---

### 9.2. Operación `actualizar`

Copia el contenido de un archivo externo dentro de un fichero registrado en `aralmac`.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0008",
  "servicio": "gesfich",
  "operacion": "actualizar",
  "datos": {
    "id_fichero": "f-0001",
    "ruta_origen": "/home/usuario/datos/numeros.txt"
  }
}
```

#### Datos requeridos

| Campo         | Tipo   | Obligatorio | Descripción                                         |
| ------------- | ------ | ----------: | --------------------------------------------------- |
| `id_fichero`  | cadena |          Sí | Fichero registrado que será actualizado.            |
| `ruta_origen` | cadena |          Sí | Ruta del archivo externo cuyo contenido se copiará. |

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0008",
  "estado": "ok",
  "resultado": {
    "id_fichero": "f-0001"
  },
  "error": null,
  "mensaje": "Fichero actualizado correctamente"
}
```

#### Errores posibles

| Código                  | Cuándo ocurre                                               |
| ----------------------- | ----------------------------------------------------------- |
| `FICHERO_NO_EXISTE`     | No existe el `id_fichero`.                                  |
| `RUTA_ORIGEN_NO_EXISTE` | No existe la ruta indicada.                                 |
| `SERVICIO_SUSPENDIDO`   | `gesfich` está suspendido y la operación no está permitida. |
| `ERROR_ALMACENAMIENTO`  | No se pudo copiar el contenido.                             |

---

### 9.3. Operación `leer`

Permite leer un fichero específico o listar todos los ficheros registrados.

La operación `leer` está permitida tanto en estado `corriendo` como en estado `suspendido`, ya que no modifica el contenido de los ficheros.

#### Leer fichero específico

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0009",
  "servicio": "gesfich",
  "operacion": "leer",
  "datos": {
    "id_fichero": "f-0001"
  }
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0009",
  "estado": "ok",
  "resultado": {
    "id_fichero": "f-0001",
    "contenido": "10\n20\n30\n"
  },
  "error": null,
  "mensaje": "Fichero leído correctamente"
}
```

#### Leer todos los ficheros

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0010",
  "servicio": "gesfich",
  "operacion": "leer",
  "datos": {}
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0010",
  "estado": "ok",
  "resultado": {
    "ficheros": [
      {
        "id_fichero": "f-0001",
        "nombre_logico": "numeros_entrada"
      },
      {
        "id_fichero": "f-0002",
        "nombre_logico": "resultado_final"
      }
    ]
  },
  "error": null,
  "mensaje": "Listado de ficheros registrados"
}
```

#### Errores posibles

| Código                 | Cuándo ocurre                         |
| ---------------------- | ------------------------------------- |
| `FICHERO_NO_EXISTE`    | Se envía un `id_fichero` inexistente. |
| `ERROR_ALMACENAMIENTO` | No se pudo acceder al contenido.      |
| `SERVICIO_TERMINADO`   | `gesfich` ya se encuentra terminado.  |

---

### 9.4. Operación `borrar`

Elimina un fichero registrado.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0011",
  "servicio": "gesfich",
  "operacion": "borrar",
  "datos": {
    "id_fichero": "f-0001"
  }
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0011",
  "estado": "ok",
  "resultado": {
    "id_fichero": "f-0001"
  },
  "error": null,
  "mensaje": "Fichero borrado correctamente"
}
```

#### Errores posibles

| Código                 | Cuándo ocurre                                               |
| ---------------------- | ----------------------------------------------------------- |
| `FICHERO_NO_EXISTE`    | No existe el `id_fichero`.                                  |
| `FICHERO_EN_USO`       | El fichero está siendo usado por un lote activo.            |
| `SERVICIO_SUSPENDIDO`  | `gesfich` está suspendido y la operación no está permitida. |
| `ERROR_ALMACENAMIENTO` | No se pudo borrar el fichero.                               |

---

### 9.5. Operaciones `suspender`, `reasumir` y `terminar`

Controlan el estado del servicio `gesfich`.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0012",
  "servicio": "gesfich",
  "operacion": "suspender",
  "datos": {}
}
```

Para `reasumir` y `terminar` se usa la misma estructura, cambiando el valor de `operacion`.

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0012",
  "estado": "ok",
  "resultado": {
    "estado_servicio": "suspendido"
  },
  "error": null,
  "mensaje": "Servicio gesfich suspendido correctamente"
}
```

#### Errores posibles

| Código               | Cuándo ocurre                                      |
| -------------------- | -------------------------------------------------- |
| `OPERACION_INVALIDA` | La transición no es válida desde el estado actual. |
| `SERVICIO_TERMINADO` | El servicio ya está terminado.                     |

---

## 10. API de `ejecutor`

`ejecutor` ejecuta procesos de lote usando programas y ficheros registrados en `aralmac`.

El ejecutor no consulta a `gesprog` ni a `gesfich`. Accede directamente a `aralmac` para validar que existan los programas y ficheros indicados.

Un lote se define como una cadena de uno o más programas conectados entre un fichero de entrada y un fichero de salida:

```text
fichero_entrada → programa_1 → programa_2 → ... → fichero_salida
```

### Estados y operaciones permitidas

| Estado de `ejecutor` | Operaciones permitidas                              |
| -------------------- | --------------------------------------------------- |
| `corriendo`          | `ejecutar`, `estado`, `matar`, `suspender`, `parar` |
| `suspendido`         | `estado`, `matar`, `reasumir`, `parar`              |
| `parar`              | `estado`, `matar`                                   |
| `terminado`          | Ninguna operación normal.                           |

En estado `suspendido`, los lotes que ya estaban en ejecución continúan corriendo. Lo que se rechaza son nuevas operaciones `ejecutar`.

---

### 10.1. Operación `ejecutar`

Inicia una ejecución de lote. La operación valida la petición, crea el lote y responde inmediatamente con un `id_lote`. La ejecución continúa en segundo plano.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0013",
  "servicio": "ejecutor",
  "operacion": "ejecutar",
  "datos": {
    "entrada": "f-0001",
    "programas": ["p-0001", "p-0002"],
    "salida": "f-0002"
  }
}
```

#### Datos requeridos

| Campo       | Tipo    | Obligatorio | Descripción                                                              |
| ----------- | ------- | ----------: | ------------------------------------------------------------------------ |
| `entrada`   | cadena  |          Sí | Identificador del fichero que será entrada del primer programa.          |
| `programas` | arreglo |          Sí | Lista ordenada de programas a ejecutar. Debe tener al menos un elemento. |
| `salida`    | cadena  |          Sí | Identificador del fichero donde se guardará la salida final.             |

#### Interpretación

```text
entrada: f-0001
programas: p-0001, p-0002
salida: f-0002

Flujo:
f-0001 → p-0001 → p-0002 → f-0002
```

#### Respuesta exitosa inmediata

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0013",
  "estado": "ok",
  "resultado": {
    "id_lote": "l-0001",
    "estado_lote": "corriendo"
  },
  "error": null,
  "mensaje": "Lote iniciado correctamente"
}
```

#### Errores posibles

| Código                  | Cuándo ocurre                                               |
| ----------------------- | ----------------------------------------------------------- |
| `FICHERO_NO_EXISTE`     | No existe el fichero de entrada o salida.                   |
| `PROGRAMA_NO_EXISTE`    | Algún programa del arreglo no existe.                       |
| `LISTA_PROGRAMAS_VACIA` | El arreglo `programas` está vacío.                          |
| `SERVICIO_SUSPENDIDO`   | El ejecutor está suspendido y no acepta nuevas ejecuciones. |
| `ERROR_EJECUCION_LOTE`  | No se pudo crear o iniciar el lote.                         |

---

### 10.2. Operación `estado`

Consulta el estado de un lote específico o lista todos los lotes.

#### Consultar lote específico

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0014",
  "servicio": "ejecutor",
  "operacion": "estado",
  "datos": {
    "id_lote": "l-0001"
  }
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0014",
  "estado": "ok",
  "resultado": {
    "id_lote": "l-0001",
    "estado_lote": "terminado_ok",
    "codigo_salida": 0
  },
  "error": null,
  "mensaje": "Estado del lote consultado correctamente"
}
```

#### Consultar todos los lotes

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0015",
  "servicio": "ejecutor",
  "operacion": "estado",
  "datos": {}
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0015",
  "estado": "ok",
  "resultado": {
    "lotes": [
      {
        "id_lote": "l-0001",
        "estado_lote": "terminado_ok",
        "codigo_salida": 0
      },
      {
        "id_lote": "l-0002",
        "estado_lote": "corriendo",
        "codigo_salida": null
      }
    ]
  },
  "error": null,
  "mensaje": "Listado de lotes consultado correctamente"
}
```

#### Estados posibles de un lote

| Estado            | Descripción                                          |
| ----------------- | ---------------------------------------------------- |
| `pendiente`       | El lote fue recibido pero aún no ha iniciado.        |
| `corriendo`       | El lote está en ejecución.                           |
| `terminado_ok`    | El lote terminó correctamente.                       |
| `terminado_error` | El lote terminó con error.                           |
| `matado`          | El lote fue terminado forzosamente mediante `matar`. |

#### Errores posibles

| Código                  | Cuándo ocurre                        |
| ----------------------- | ------------------------------------ |
| `LOTE_NO_EXISTE`        | Se envía un `id_lote` que no existe. |
| `ERROR_CONSULTA_ESTADO` | No se pudo consultar el estado.      |

---

### 10.3. Operación `matar`

Detiene forzosamente un lote en ejecución.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0016",
  "servicio": "ejecutor",
  "operacion": "matar",
  "datos": {
    "id_lote": "l-0001"
  }
}
```

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0016",
  "estado": "ok",
  "resultado": {
    "id_lote": "l-0001",
    "estado_lote": "matado"
  },
  "error": null,
  "mensaje": "Lote detenido correctamente"
}
```

#### Observación

Si el lote ya había escrito datos en el fichero de salida antes de ser matado, el fichero de salida puede quedar con contenido parcial.

#### Errores posibles

| Código                 | Cuándo ocurre                          |
| ---------------------- | -------------------------------------- |
| `LOTE_NO_EXISTE`       | No existe el `id_lote`.                |
| `LOTE_YA_TERMINADO`    | El lote ya terminó y no puede matarse. |
| `ERROR_EJECUCION_LOTE` | No se pudo terminar el proceso.        |

---

### 10.4. Operaciones `suspender`, `reasumir` y `parar`

Controlan el estado del servicio `ejecutor`.

#### Petición

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0017",
  "servicio": "ejecutor",
  "operacion": "suspender",
  "datos": {}
}
```

Para `reasumir` y `parar` se usa la misma estructura, cambiando el valor de `operacion`.

#### Respuesta exitosa para `suspender`

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0017",
  "estado": "ok",
  "resultado": {
    "estado_servicio": "suspendido"
  },
  "error": null,
  "mensaje": "Ejecutor suspendido correctamente"
}
```

#### Respuesta exitosa para `parar`

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0018",
  "estado": "ok",
  "resultado": {
    "estado_servicio": "parar",
    "procesos_activos": 2
  },
  "error": null,
  "mensaje": "Ejecutor en modo de parada ordenada"
}
```

La operación `parar` hace que el ejecutor deje de aceptar nuevas ejecuciones y espere a que terminen los lotes activos. Cuando no quedan procesos activos, el ejecutor puede pasar a `terminado`.

#### Errores posibles

| Código               | Cuándo ocurre                                      |
| -------------------- | -------------------------------------------------- |
| `OPERACION_INVALIDA` | La transición no es válida desde el estado actual. |
| `SERVICIO_TERMINADO` | El ejecutor ya está terminado.                     |

---

## 11. API de `ctrllt`

`ctrllt` recibe peticiones JSON y las redirige al servicio correspondiente según el campo `servicio`.

### 11.1. Petición recibida por `ctrllt`

`ctrllt` recibe el mismo formato estándar de petición usado por los demás servicios.

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0019",
  "servicio": "gesfich",
  "operacion": "leer",
  "datos": {
    "id_fichero": "f-0001"
  }
}
```

### 11.2. Enrutamiento

| Campo `servicio` | Acción de `ctrllt`                                           |
| ---------------- | ------------------------------------------------------------ |
| `gesprog`        | Reenvía la petición a `gesprog`.                             |
| `gesfich`        | Reenvía la petición a `gesfich`.                             |
| `ejecutor`       | Reenvía la petición a `ejecutor`.                            |
| `ctrllt`         | Atiende la operación directamente si corresponde al control. |

### 11.3. Respuesta devuelta por `ctrllt`

`ctrllt` devuelve al cliente la respuesta generada por el servicio correspondiente, conservando el mismo `id_cliente` e `id_peticion`.

### 11.4. Operación `terminar`

`ctrllt` puede recibir una operación de control para terminar.

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0020",
  "servicio": "ctrllt",
  "operacion": "terminar",
  "datos": {}
}
```

Al recibir `terminar`, `ctrllt` interpreta la operación como una orden global de apagado del sistema. Por tanto, debe terminar también los servicios `gesprog`, `gesfich` y `ejecutor`. Para `gesprog` y `gesfich` solicita la operación `terminar`; para `ejecutor` solicita su cierre ordenado mediante `parar`, de forma que deje de aceptar nuevas ejecuciones y finalice cuando no existan procesos activos. Luego `ctrllt` finaliza su propia ejecución.

#### Respuesta exitosa

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0020",
  "estado": "ok",
  "resultado": {
    "estado_servicio": "terminado"
  },
  "error": null,
  "mensaje": "Control terminado correctamente"
}
```

---

## 12. Resumen general de operaciones

| Servicio   | Operación    | Contenido de `datos`                                                          |
| ---------- | ------------ | ----------------------------------------------------------------------------- |
| `gesprog`  | `guardar`    | `ejecutable`, `argumentos`, `ambiente`, `descripcion` opcional                |
| `gesprog`  | `leer`       | `id_programa` o `{}`                                                          |
| `gesprog`  | `actualizar` | `id_programa`, `ejecutable`, `argumentos`, `ambiente`, `descripcion` opcional |
| `gesprog`  | `borrar`     | `id_programa`                                                                 |
| `gesprog`  | `suspender`  | `{}`                                                                          |
| `gesprog`  | `reasumir`   | `{}`                                                                          |
| `gesprog`  | `terminar`   | `{}`                                                                          |
| `gesfich`  | `crear`      | `nombre_logico` opcional                                                      |
| `gesfich`  | `leer`       | `id_fichero` o `{}`                                                           |
| `gesfich`  | `actualizar` | `id_fichero`, `ruta_origen`                                                   |
| `gesfich`  | `borrar`     | `id_fichero`                                                                  |
| `gesfich`  | `suspender`  | `{}`                                                                          |
| `gesfich`  | `reasumir`   | `{}`                                                                          |
| `gesfich`  | `terminar`   | `{}`                                                                          |
| `ejecutor` | `ejecutar`   | `entrada`, `programas`, `salida`                                              |
| `ejecutor` | `estado`     | `id_lote` o `{}`                                                              |
| `ejecutor` | `matar`      | `id_lote`                                                                     |
| `ejecutor` | `suspender`  | `{}`                                                                          |
| `ejecutor` | `reasumir`   | `{}`                                                                          |
| `ejecutor` | `parar`      | `{}`                                                                          |
| `ctrllt`   | `terminar`   | `{}`                                                                          |

---

## 13. Catálogo general de errores

Además de los errores indicados en cada operación, se define el siguiente catálogo general de códigos de error.

| Código                       | Descripción                                                                    | Servicios que lo usan            |
| ---------------------------- | ------------------------------------------------------------------------------ | -------------------------------- |
| `CAMPO_OBLIGATORIO_FALTANTE` | Falta un campo requerido en `datos`.                                           | `gesprog`, `gesfich`, `ejecutor` |
| `SERVICIO_INVALIDO`          | El valor de `servicio` no corresponde a un servicio válido.                    | `ctrllt`                         |
| `OPERACION_INVALIDA`         | La operación no existe para el servicio o no es válida desde el estado actual. | Todos                            |
| `PROGRAMA_NO_EXISTE`         | No existe el programa indicado por `id_programa`.                              | `gesprog`, `ejecutor`            |
| `FICHERO_NO_EXISTE`          | No existe el fichero indicado por `id_fichero`.                                | `gesfich`, `ejecutor`            |
| `LOTE_NO_EXISTE`             | No existe el lote indicado por `id_lote`.                                      | `ejecutor`                       |
| `EJECUTABLE_NO_EXISTE`       | La ruta del ejecutable no existe.                                              | `gesprog`                        |
| `EJECUTABLE_INVALIDO`        | El archivo no puede ejecutarse o no es válido.                                 | `gesprog`                        |
| `RUTA_ORIGEN_NO_EXISTE`      | La ruta indicada para cargar contenido no existe.                              | `gesfich`                        |
| `LISTA_PROGRAMAS_VACIA`      | El arreglo `programas` está vacío.                                             | `ejecutor`                       |
| `SERVICIO_SUSPENDIDO`        | El servicio está suspendido y la operación solicitada no está permitida.       | `gesprog`, `gesfich`, `ejecutor` |
| `SERVICIO_TERMINADO`         | El servicio ya se encuentra terminado.                                         | Todos                            |
| `ERROR_ALMACENAMIENTO`       | No se pudo leer, escribir, copiar o borrar información en `aralmac`.           | `gesprog`, `gesfich`, `ejecutor` |
| `PROGRAMA_EN_USO`            | Se intenta borrar un programa usado por un lote activo.                        | `gesprog`                        |
| `FICHERO_EN_USO`             | Se intenta borrar un fichero usado por un lote activo.                         | `gesfich`                        |
| `ERROR_EJECUCION_LOTE`       | Error al crear, ejecutar, matar o controlar un lote.                           | `ejecutor`                       |
| `ERROR_CONSULTA_ESTADO`      | No se pudo consultar el estado de un lote.                                     | `ejecutor`                       |
| `LOTE_YA_TERMINADO`          | Se intenta matar un lote que ya terminó.                                       | `ejecutor`                       |

---

## 14. Consideraciones sobre `aralmac`

`aralmac` es el área de almacenamiento del sistema. Puede implementarse como un directorio del sistema de archivos.

Estructura sugerida:

```text
aralmac/
├── programas/
│   ├── p-0001/
│   │   ├── ejecutable
│   │   └── metadata.json
│   └── p-0002/
│       ├── ejecutable
│       └── metadata.json
├── ficheros/
│   ├── f-0001
│   └── f-0002
├── lotes/
│   └── l-0001.json
└── estado/
    ├── gesprog.json
    ├── gesfich.json
```

*Nota:* El directorio `lotes/` puede contener información temporal de ejecuciones, pero no es esencial para la persistencia de identificadores. El ejecutor no necesita restaurar lotes activos tras un reinicio.

Responsabilidades:

| Componente | Uso de `aralmac`                                                                             |
| ---------- | -------------------------------------------------------------------------------------------- |
| `gesprog`  | Guarda, consulta, actualiza y borra programas.                                               |
| `gesfich`  | Crea, consulta, actualiza y borra ficheros.                                                  |
| `ejecutor` | Consulta programas y ficheros para ejecutar lotes; escribe resultados en ficheros de salida. |

### 14.1. Persistencia de identificadores

Los identificadores de programas y ficheros deben mantenerse únicos incluso si un servicio se reinicia. Para ello, `gesprog` y `gesfich` conservan un archivo de estado dentro de `aralmac/estado/`.

Ejemplo:

```text
aralmac/estado/gesprog.json   → { "ultimo_id": 3 }
aralmac/estado/gesfich.json   → { "ultimo_id": 5 }
```

Cuando un servicio inicia, lee su archivo de estado y genera el siguiente identificador a partir del último valor almacenado.

El `id_lote`, en cambio, **no persiste**: cada vez que el ejecutor arranca, la numeración comienza desde `l-0001`. Los lotes activos se pierden al reiniciar el ejecutor, lo cual es aceptable por su carácter efímero.

Ejemplo:

```text
Si aralmac/estado/gesfich.json contiene { "ultimo_id": 5 },
el siguiente fichero creado será f-0006.
```

---

## 15. Orden de arranque recomendado

El orden de arranque recomendado es:

1. Crear o verificar la existencia de `aralmac`.
2. Iniciar `gesfich`.
3. Iniciar `gesprog`.
4. Iniciar `ejecutor`.
5. Iniciar `ctrllt`, si se usará el modo pasarela.
6. Ejecutar el cliente.

Este orden permite que los servicios creen sus tuberías antes de que el cliente o `ctrllt` intenten conectarse a ellas.

Ejemplo de arranque en Linux:

```bash
./gesfich -f /tmp/lotes-gesfich-req -b /tmp/lotes-gesfich-res -x ./aralmac &
./gesprog -p /tmp/lotes-gesprog-req -c /tmp/lotes-gesprog-res -x ./aralmac &
./ejecutor -e /tmp/lotes-ejecutor-req -d /tmp/lotes-ejecutor-res -x ./aralmac &
./ctrllt -c /tmp/lotes-ctrllt-req -a /tmp/lotes-ctrllt-res \
         -f /tmp/lotes-gesfich-req -b /tmp/lotes-gesfich-res \
         -p /tmp/lotes-gesprog-req -s /tmp/lotes-gesprog-res \
         -e /tmp/lotes-ejecutor-req -d /tmp/lotes-ejecutor-res &
```

Si se usa el modo directo, el cliente puede conectarse directamente a las tuberías del servicio correspondiente sin iniciar `ctrllt`.

---

## 16. Máquinas de estado

### 16.1. `ctrllt`

```text
inicio → corriendo → terminado
```

| Estado      | Descripción                            |
| ----------- | -------------------------------------- |
| `inicio`    | Estado inicial.                        |
| `corriendo` | Recibe, analiza y redirige peticiones. |
| `terminado` | El servicio deja de ejecutarse.        |

| Operación           | Transición              |
| ------------------- | ----------------------- |
| inicio del servicio | `inicio → corriendo`    |
| `terminar`          | `corriendo → terminado` |

---

### 16.2. `gesprog`

```text
inicio → corriendo
corriendo → suspendido
suspendido → corriendo
corriendo → terminado
suspendido → terminado
```

| Estado       | Operaciones permitidas                                             |
| ------------ | ------------------------------------------------------------------ |
| `corriendo`  | `guardar`, `leer`, `actualizar`, `borrar`, `suspender`, `terminar` |
| `suspendido` | `leer`, `reasumir`, `terminar`                                     |
| `terminado`  | Ninguna operación normal.                                          |

| Operación           | Transición                         |
| ------------------- | ---------------------------------- |
| inicio del servicio | `inicio → corriendo`               |
| `suspender`         | `corriendo → suspendido`           |
| `reasumir`          | `suspendido → corriendo`           |
| `terminar`          | `corriendo/suspendido → terminado` |

---

### 16.3. `gesfich`

```text
inicio → corriendo
corriendo → suspendido
suspendido → corriendo
corriendo → terminado
suspendido → terminado
```

| Estado       | Operaciones permitidas                                           |
| ------------ | ---------------------------------------------------------------- |
| `corriendo`  | `crear`, `leer`, `actualizar`, `borrar`, `suspender`, `terminar` |
| `suspendido` | `leer`, `reasumir`, `terminar`                                   |
| `terminado`  | Ninguna operación normal.                                        |

| Operación           | Transición                         |
| ------------------- | ---------------------------------- |
| inicio del servicio | `inicio → corriendo`               |
| `suspender`         | `corriendo → suspendido`           |
| `reasumir`          | `suspendido → corriendo`           |
| `terminar`          | `corriendo/suspendido → terminado` |

---

### 16.4. `ejecutor`

```text
inicio → corriendo
corriendo → suspendido
suspendido → corriendo
corriendo → parar
suspendido → parar
parar → terminado   cuando procesos_activos = 0
```

| Estado       | Operaciones permitidas                              | Descripción                                                                    |
| ------------ | --------------------------------------------------- | ------------------------------------------------------------------------------ |
| `corriendo`  | `ejecutar`, `estado`, `matar`, `suspender`, `parar` | Acepta nuevas ejecuciones y controla lotes activos.                            |
| `suspendido` | `estado`, `matar`, `reasumir`, `parar`              | No acepta nuevas ejecuciones; los lotes activos continúan corriendo.           |
| `parar`      | `estado`, `matar`                                   | Deja de aceptar nuevas ejecuciones y espera a que no existan procesos activos. |
| `terminado`  | Ninguna operación normal.                           | Estado final.                                                                  |

| Operación              | Transición                     |
| ---------------------- | ------------------------------ |
| inicio del servicio    | `inicio → corriendo`           |
| `suspender`            | `corriendo → suspendido`       |
| `reasumir`             | `suspendido → corriendo`       |
| `parar`                | `corriendo/suspendido → parar` |
| `procesos_activos = 0` | `parar → terminado`            |

Diferencia entre `matar` y `parar`:

| Operación | Alcance           | Descripción                                          |
| --------- | ----------------- | ---------------------------------------------------- |
| `matar`   | Lote específico   | Detiene forzosamente el lote indicado por `id_lote`. |
| `parar`   | Servicio ejecutor | Inicia el cierre ordenado del ejecutor.              |

---

## 17. Ejemplo completo de flujo

### 17.1. Registrar programa

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0001",
  "servicio": "gesprog",
  "operacion": "guardar",
  "datos": {
    "ejecutable": "/home/usuario/programas/sumar.py",
    "argumentos": [],
    "ambiente": {},
    "descripcion": "Suma los números recibidos por entrada estándar"
  }
}
```

Respuesta:

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0001",
  "estado": "ok",
  "resultado": {
    "id_programa": "p-0001"
  },
  "error": null,
  "mensaje": "Programa registrado correctamente"
}
```

### 17.2. Crear fichero de entrada

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0002",
  "servicio": "gesfich",
  "operacion": "crear",
  "datos": {
    "nombre_logico": "numeros_entrada"
  }
}
```

Respuesta:

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0002",
  "estado": "ok",
  "resultado": {
    "id_fichero": "f-0001"
  },
  "error": null,
  "mensaje": "Fichero creado correctamente"
}
```

### 17.3. Actualizar fichero de entrada

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0003",
  "servicio": "gesfich",
  "operacion": "actualizar",
  "datos": {
    "id_fichero": "f-0001",
    "ruta_origen": "/home/usuario/datos/numeros.txt"
  }
}
```

Respuesta:

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0003",
  "estado": "ok",
  "resultado": {
    "id_fichero": "f-0001"
  },
  "error": null,
  "mensaje": "Fichero actualizado correctamente"
}
```

### 17.4. Crear fichero de salida

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0004",
  "servicio": "gesfich",
  "operacion": "crear",
  "datos": {
    "nombre_logico": "resultado_final"
  }
}
```

Respuesta:

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0004",
  "estado": "ok",
  "resultado": {
    "id_fichero": "f-0002"
  },
  "error": null,
  "mensaje": "Fichero creado correctamente"
}
```

### 17.5. Ejecutar lote

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0005",
  "servicio": "ejecutor",
  "operacion": "ejecutar",
  "datos": {
    "entrada": "f-0001",
    "programas": ["p-0001"],
    "salida": "f-0002"
  }
}
```

Respuesta inmediata:

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0005",
  "estado": "ok",
  "resultado": {
    "id_lote": "l-0001",
    "estado_lote": "corriendo"
  },
  "error": null,
  "mensaje": "Lote iniciado correctamente"
}
```

### 17.6. Consultar estado del lote

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0006",
  "servicio": "ejecutor",
  "operacion": "estado",
  "datos": {
    "id_lote": "l-0001"
  }
}
```

Respuesta:

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0006",
  "estado": "ok",
  "resultado": {
    "id_lote": "l-0001",
    "estado_lote": "terminado_ok",
    "codigo_salida": 0
  },
  "error": null,
  "mensaje": "Estado del lote consultado correctamente"
}
```

### 17.7. Leer fichero de salida

```json
{
  "tipo_mensaje": "peticion",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0007",
  "servicio": "gesfich",
  "operacion": "leer",
  "datos": {
    "id_fichero": "f-0002"
  }
}
```

Respuesta:

```json
{
  "tipo_mensaje": "respuesta",
  "id_cliente": "cli-0001",
  "id_peticion": "req-0007",
  "estado": "ok",
  "resultado": {
    "id_fichero": "f-0002",
    "contenido": "60\n"
  },
  "error": null,
  "mensaje": "Fichero leído correctamente"
}
```

---

## 18. Conclusión

Este documento define la API JSON del sistema Ejecutor de Lotes. La API permite administrar programas, ficheros y procesos de lote mediante mensajes estructurados, identificadores y respuestas estandarizadas.

El diseño contempla tanto la comunicación directa entre cliente y servicios como el uso de `ctrllt` como pasarela. Además, mantiene diferencias claras entre Linux y Windows en el uso de tuberías nombradas, sin alterar el formato de los mensajes JSON.

La API queda organizada alrededor de cuatro servicios principales:

```text
gesprog  → programas
gesfich  → ficheros
ejecutor → lotes
ctrllt   → control y pasarela
```