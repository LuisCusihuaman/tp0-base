## Protocolo de Comunicación Cliente-Servidor

### Resumen

Este documento describe el protocolo de comunicación entre el cliente y el servidor, utilizado para enviar y recibir
paquetes. El protocolo maneja la serialización de datos, la separación de responsabilidades entre el modelo de dominio y
la capa de comunicación, y el correcto manejo de sockets, evitando errores comunes como _short read_ y _short write_.

### Estructura del Mensaje

| Campo               | Tamaño   | Descripción                                                                                  |
|---------------------|----------|----------------------------------------------------------------------------------------------|
| **Header**          | 4 bytes  | Longitud total del mensaje (incluye el tamaño del tipo de mensaje y la longitud del cuerpo). |
| **Tipo de mensaje** | 1 byte   | Indica el tipo de mensaje según la tabla de tipos de mensajes.                               |
| **Cuerpo**          | Variable | Contiene los datos serializados según el tipo de mensaje.                                    |

### Tipos de Mensaje

| Nombre        | Valor | Descripción                                  |
|---------------|-------|----------------------------------------------|
| `MSG_SUCCESS` | 0x00  | Indica que la operación fue exitosa.         |
| `MSG_ERROR`   | 0x01  | Indica que ocurrió un error en la operación. |
| `MSG_BET`     | 0x10  | Mensaje que contiene una apuesta.            |
| `MSG_BATCH`   | 0x11  | Contiene un lote de apuestas (`Bet`).        |
| `MSG_ECHO`    | 0x12  | Mensaje de echo para pruebas o diagnósticos. |

### Códigos de Respuesta para `MSG_SUCCESS`

| Código de Éxito           | Valor | Descripción                     |
|---------------------------|-------|---------------------------------|
| `SUCCESS_BATCH_PROCESSED` | 0x01  | Lote procesado exitosamente.    |
| `SUCCESS_BET_PROCESSED`   | 0x02  | Apuesta procesada exitosamente. |

### Códigos de Respuesta para `MSG_ERROR`

| Código de Error           | Valor | Descripción                   |
|---------------------------|-------|-------------------------------|
| `ERROR_BATCH_FAILED`      | 0x01  | Fallo al procesar el lote.    |
| `ERROR_BET_FAILED`        | 0x02  | Fallo al procesar la apuesta. |
| `ERROR_MALFORMED_MESSAGE` | 0x03  | El mensaje está mal formado.  |
| `ERROR_INVALID_MESSAGE`   | 0x04  | El mensaje es inválido.       |

### Formato del Mensaje de Respuesta (Longitud Fija)

Los mensajes de respuesta del servidor al cliente tienen un tamaño fijo. La estructura es la siguiente:

| Campo               | Tipo     | Tamaño  | Descripción                                                          |
|---------------------|----------|---------|----------------------------------------------------------------------|
| **Header**          | `uint32` | 4 bytes | Longitud total del mensaje (incluye el tipo de mensaje y el cuerpo). |
| **Tipo de mensaje** | `uint8`  | 1 byte  | Tipo de mensaje (`MSG_SUCCESS` o `MSG_ERROR`).                       |
| **Código**          | `uint8`  | 1 byte  | Código específico de éxito o error.                                  |

### Ejemplo de Mensaje de Respuesta `MSG_SUCCESS`

```
Header (4 bytes): 0x00 0x00 0x00 0x06  // Longitud total del mensaje (6 bytes)
Tipo de mensaje (1 byte): 0x00         // Tipo: MSG_SUCCESS
Código de Éxito (1 byte): 0x01         // Código: SUCCESS_BATCH_PROCESSED
```

### Ejemplo de Mensaje de Respuesta `MSG_ERROR`

```
Header (4 bytes): 0x00 0x00 0x00 0x06  // Longitud total del mensaje (6 bytes)
Tipo de mensaje (1 byte): 0x01         // Tipo: MSG_ERROR
Código de Error (1 byte): 0x02         // Código: ERROR_BET_FAILED
```

### Formato del Mensaje `MSG_BET`

| Cuerpo       | Tipo     | Tamaño   | Descripción                                                 |
|--------------|----------|----------|-------------------------------------------------------------|
| `Agency`     | `uint32` | 4 bytes  | Identificador de la agencia.                                |
| `First Name` | Cadena   | Variable | Longitud (4 bytes `uint32`) seguida por la cadena en UTF-8. |
| `Last Name`  | Cadena   | Variable | Longitud (4 bytes `uint32`) seguida por la cadena en UTF-8. |
| `Document`   | `uint32` | 4 bytes  | Número de documento (DNI).                                  |
| `Birth Date` | Cadena   | 10 bytes | Fecha de nacimiento en formato "YYYY-MM-DD".                |
| `Number`     | `uint32` | 4 bytes  | Número asociado a la apuesta.                               |

### Ejemplo de Mensaje `MSG_BET`

```
Header (4 bytes): 0x00 0x00 0x00 0x2B  // Longitud total del mensaje (43 bytes)
Tipo de mensaje (1 byte): 0x10         // Tipo: MSG_BET
Cuerpo:
    Agency (4 bytes): 0x00 0x00 0x00 0x01
    First Name (13 bytes): 0x00 0x00 0x00 0x04 0x4A 0x6F 0x68 0x6E  // Longitud + "John"
    Last Name (14 bytes): 0x00 0x00 0x00 0x05 0x44 0x6F 0x65 0x73  // Longitud + "Does"
    Document (4 bytes): 0x01 0x23 0x45 0x67  // DNI: 19088743
    Birth Date (10 bytes): 0x32 0x30 0x30 0x30 0x2D 0x31 0x30 0x2D 0x30 0x31  // "2000-10-01"
    Number (4 bytes): 0x00 0x00 0x00 0x0A
```

### Formato del Mensaje `MSG_BATCH`

El mensaje `MSG_BATCH` agrupa múltiples mensajes `MSG_BET` en un solo envío. Este tipo de mensaje es útil para reducir
la sobrecarga de comunicación cuando se desea enviar múltiples apuestas al mismo tiempo.

#### Estructura del Mensaje `MSG_BATCH`

| Cuerpo      | Tipo     | Tamaño   | Descripción                                                                         |
|-------------|----------|----------|-------------------------------------------------------------------------------------|
| `Bet Count` | `uint32` | 4 bytes  | Número total de apuestas en el lote.                                                |
| `Bets`      | Cadena   | Variable | Serie de mensajes `MSG_BET`, cada uno con su propio formato descrito anteriormente. |

### Ejemplo de Mensaje `MSG_BATCH`

Imaginemos que queremos enviar un lote de dos apuestas (`Bet`). La estructura del mensaje sería la siguiente:

```
Header (4 bytes): 0x00 0x00 0x00 0x57  // Longitud total del mensaje (87 bytes)
Tipo de mensaje (1 byte): 0x11         // Tipo: MSG_BATCH
Cuerpo:
    Bet Count (4 bytes): 0x00 0x00 0x00 0x02  // Número de apuestas: 2
    Bets:
        Apuesta 1:
            Agency (4 bytes): 0x00 0x00 0x00 0x01
            First Name (13 bytes): 0x00 0x00 0x00 0x04 0x4A 0x6F 0x68 0x6E  // Longitud + "John"
            Last Name (14 bytes): 0x00 0x00 0x00 0x05 0x44 0x6F 0x65 0x73  // Longitud + "Does"
            Document (4 bytes): 0x01 0x23 0x45 0x67  // DNI: 19088743
            Birth Date (10 bytes): 0x32 0x30 0x30 0x30 0x2D 0x31 0x30 0x2D 0x30 0x31  // "2000-10-01"
            Number (4 bytes): 0x00 0x00 0x00 0x0A
        Apuesta 2:
            Agency (4 bytes): 0x00 0x00 0x00 0x02
            First Name (13 bytes): 0x00 0x00 0x00 0x03 0x45 0x6C 0x6C 0x61  // Longitud + "Ella"
            Last Name (14 bytes): 0x00 0x00 0x00 0x03 0x4C 0x65 0x65 0x65  // Longitud + "Lee"
            Document (4 bytes): 0x02 0x34 0x56 0x78  // DNI: 3735928559
            Birth Date (10 bytes): 0x32 0x30 0x30 0x31 0x2D 0x30 0x39 0x2D 0x32 0x30  // "2001-09-20"
            Number (4 bytes): 0x00 0x00 0x00 0x14
```
