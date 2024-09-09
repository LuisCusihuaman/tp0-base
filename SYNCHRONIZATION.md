# Tabla de Contenido

1. [Mecanismos de Sincronización en el Servidor](#mecanismos-de-sincronización-en-el-servidor)  
   1.1. [Uso de Hilos (Threads)](#1-uso-de-hilos-threads)  
   1.2. [Consideración sobre el GIL (Global Interpreter Lock)](#consideración-sobre-el-gil-global-interpreter-lock)  
   1.3. [Bloqueos (Locks)](#2-bloqueos-locks)  
   1.4. [Barrera (Barrier)](#3-barrera-barrier)  
   1.5. [Manejo de Errores](#4-manejo-de-errores)

2. [Mecanismos de Sincronización en el Cliente](#mecanismos-de-sincronización-en-el-cliente)  
   2.1. [Hilos Concurrentes para Envío de Datos](#1-hilos-concurrentes-para-envío-de-datos)  
   2.2. [Canales (Channels) para Comunicación entre Goroutines](#2-canales-channels-para-comunicación-entre-goroutines)  
   2.3. [Mecanismo de Reintento para Consultar Ganadores](#3-mecanismo-de-reintento-para-consultar-ganadores)

3. [Conclusión](#conclusión)

## Mecanismos de Sincronización en el Servidor

Este documento explica los mecanismos de sincronización que se utilizaron en la implementación del servidor, el cual
permite manejar múltiples conexiones de clientes al mismo tiempo de forma concurrente. La sincronización es necesaria
para evitar problemas como condiciones de carrera y asegurar que los datos compartidos se mantengan consistentes.

## 1. **Uso de Hilos (Threads)**

El servidor utiliza el módulo `threading` de Python para crear hilos y manejar múltiples conexiones de clientes de
manera simultánea. Cada vez que un cliente se conecta al servidor, se crea un nuevo hilo para gestionar su comunicación,
lo que permite que el servidor continúe aceptando otras conexiones sin bloquearse.

### Puntos Clave:

- **Hilo por cliente**: Cada cliente tiene su propio hilo que maneja la comunicación con el servidor.
- **Cierre ordenado**: Utilizamos un objeto `threading.Event` llamado `_shutdown_event` para cerrar el servidor de
  manera segura, esperando que todos los hilos de clientes terminen antes de apagar el servidor.

### **Consideración sobre el GIL (Global Interpreter Lock)**:

En Python, existe una limitación conocida como el **Global Interpreter Lock** o GIL. El GIL es un mecanismo que permite
que solo un hilo se ejecute a la vez en programas que usan CPython (la implementación más común de Python). Esto puede
ser una limitación cuando trabajamos con tareas que requieren mucha CPU, ya que, aunque usemos varios hilos, solo uno
estará ejecutándose a la vez. Sin embargo, en este caso no es un gran problema porque nuestro servidor maneja
principalmente **operaciones de entrada/salida (I/O-bound)**, como leer y escribir datos de los sockets, que no se ven tan afectadas por el GIL.

## 2. **Bloqueos (Locks)**

Para evitar que varios hilos accedan a los mismos datos al mismo tiempo y provoquen errores o inconsistencias, se
usa `threading.Lock`. Esto asegura que solo un hilo pueda modificar los datos compartidos en un momento dado.

### Ejemplo:

- **Registro de apuestas**: En la clase `LotteryManager`, se utiliza un bloqueo (`self.lock`) para proteger la sección
  del código donde se registran apuestas o se realiza el sorteo. Esto garantiza que solo un hilo a la vez pueda acceder
  a los datos que se guardan en el sistema.

### Puntos Clave:

- **Protección de recursos compartidos**: El bloqueo asegura que solo un hilo a la vez pueda registrar apuestas o
  realizar el sorteo.
- **Operaciones atómicas**: Usar un bloqueo permite que ciertos fragmentos de código se ejecuten de forma completa (sin
  interrupciones de otros hilos), evitando posibles errores.

## 3. **Barrera (Barrier)**

Una barrera (`threading.Barrier`) se utiliza en el `LotteryManager` para sincronizar las notificaciones de diferentes
agencias. La barrera asegura que el sorteo solo se realice una vez que todas las agencias hayan enviado su notificación.

### Ejemplo:

- **Sincronización de agencias**: El objeto `self.agency_barrier` obliga a que todas las agencias se queden esperando
  hasta que todas hayan enviado su notificación. Solo cuando todas han notificado, se ejecuta el sorteo.

### Puntos Clave:

- **Sincronización de múltiples hilos**: La barrera asegura que el sorteo no se realice hasta que todas las agencias
  hayan enviado su notificación, lo que garantiza que el proceso se ejecute de manera correcta y ordenada.

## 4. **Manejo de Errores**

Además de la sincronización, se incluye un manejo de errores que permite que el servidor continúe funcionando de manera
segura ante situaciones inesperadas, como cuando una agencia no puede enviar su notificación o cuando ocurre un error de
conexión.

### Puntos Clave:

- **Errores en la barrera**: Si una barrera se rompe (por ejemplo, si una agencia falla), el servidor captura ese error
  y continúa funcionando.
- **Errores de conexión**: El servidor maneja los errores de conexión cerrando los sockets de los clientes que fallan y
  permitiendo que otros hilos continúen funcionando.

## Mecanismos de Sincronización en el Cliente

El cliente está diseñado para manejar varias tareas simultáneamente, como leer las apuestas, enviarlas en lotes,
notificar al servidor y consultar los ganadores. Para lograr esto de manera eficiente, se utilizan varios mecanismos de
concurrencia y control de reintentos.

### 1. **Hilos Concurrentes para Envío de Datos**

El cliente utiliza **goroutines** (en el caso de Go) para ejecutar tareas en paralelo, lo que permite realizar varias
operaciones al mismo tiempo sin bloquear el resto del flujo. Por ejemplo:

- **Lectura de apuestas**: Se hace en un hilo separado para no interrumpir el procesamiento de lotes.
- **Envío de lotes y notificaciones**: Se realiza en hilos paralelos, manejando varias conexiones con el servidor de
  manera concurrente.

### 2. **Canales (Channels) para Comunicación entre Goroutines**

Se utilizan **canales** (en Go) para pasar datos entre las diferentes goroutines de manera segura, evitando problemas de
sincronización:

- Un canal para las apuestas (`betChan`) que se llenan desde un archivo CSV.
- Un canal para manejar las respuestas del servidor (`retryChan`), el cual permite recibir la lista de ganadores o
  enviar señales para reintentar la consulta en caso de fallo.

### 3. **Mecanismo de Reintento para Consultar Ganadores**

El cliente incluye un mecanismo para **reintentar** la consulta de ganadores si la primera consulta no recibe una
respuesta. Esto se controla mediante un bucle que reintenta hasta 5 veces, con intervalos de espera entre cada intento.
Este enfoque asegura que, en caso de problemas de red o latencia, el cliente pueda seguir intentando obtener una
respuesta sin bloquear el resto de la operación.

## Conclusión

El uso de hilos, bloqueos y barreras garantiza que el servidor y el cliente puedan manejar múltiples operaciones de
manera concurrente sin causar problemas en los datos. Estos mecanismos de sincronización son esenciales para mantener el
funcionamiento correcto y evitar errores cuando varios hilos acceden a recursos compartidos.
