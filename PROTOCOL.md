## Protocolo de Comunicación Cliente-Servidor

### Resumen

Este documento describe el protocolo de comunicación entre el cliente y el servidor, utilizado para enviar y recibir
paquetes. El protocolo maneja la serialización de datos, la separación de responsabilidades entre el modelo de dominio y
la capa de comunicación, y el correcto manejo de sockets, evitando errores comunes como _short read_ y _short write_.

### Estructura del Mensaje

| Campo               | Tamaño   | Descripción                                                          |
|---------------------|----------|----------------------------------------------------------------------|
| **Header**          | 4 bytes  | Longitud total del mensaje (incluye el tipo de mensaje y el cuerpo). |
| **Tipo de mensaje** | 1 byte   | Indica el tipo de mensaje según la tabla de tipos de mensajes.       |
| **Cuerpo**          | Variable | Contiene los datos serializados según el tipo de mensaje.            |

### Tipos de Mensaje

| Nombre             | Valor | Descripción                                  |
|--------------------|-------|----------------------------------------------|
| `MSG_SUCCESS`      | 0x00  | Indica que la operación fue exitosa.         |
| `MSG_BET`          | 0x01  | Mensaje que contiene una apuesta.            |
| `MSG_ECHO`         | 0x02  | Mensaje de echo para pruebas o diagnósticos. |
| `MSG_ERROR`        | 0x03  | Mensaje que indica un error genérico.        |
| `REJECT_MALFORMED` | 0x04  | Indica que el mensaje está mal formado.      |
| `REJECT_INVALID`   | 0x05  | Indica que el mensaje es inválido.           |

### Formato del Mensaje `MSG_BET`

| Campo        | Tipo     | Tamaño   | Descripción                                                 |
|--------------|----------|----------|-------------------------------------------------------------|
| `Agency`     | `uint32` | 4 bytes  | Identificador de la agencia.                                |
| `First Name` | Cadena   | Variable | Longitud (4 bytes `uint32`) seguida por la cadena en UTF-8. |
| `Last Name`  | Cadena   | Variable | Longitud (4 bytes `uint32`) seguida por la cadena en UTF-8. |
| `Document`   | Cadena   | Variable | Longitud (4 bytes `uint32`) seguida por la cadena en UTF-8. |
| `Birth Date` | Cadena   | 10 bytes | Fecha de nacimiento en formato "YYYY-MM-DD".                |
| `Number`     | `uint32` | 4 bytes  | Número asociado a la apuesta.                               |

### Ejemplo de Mensaje `MSG_BET`

```
Header (4 bytes): 0x00 0x00 0x00 0x2F  // Longitud total del mensaje (47 bytes)
Tipo de mensaje (1 byte): 0x01         // Tipo: MSG_BET
Cuerpo:
    Agency (4 bytes): 0x00 0x00 0x00 0x01
    First Name (13 bytes): 0x00 0x00 0x00 0x04 0x4A 0x6F 0x68 0x6E  // Longitud + "John"
    Last Name (14 bytes): 0x00 0x00 0x00 0x05 0x44 0x6F 0x65 0x73  // Longitud + "Does"
    Document (12 bytes): 0x00 0x00 0x00 0x03 0x31 0x32 0x33  // Longitud + "123"
    Birth Date (10 bytes): 0x32 0x30 0x30 0x30 0x2D 0x31 0x30 0x2D 0x30 0x31  // "2000-10-01"
    Number (4 bytes): 0x00 0x00 0x00 0x0A
```
