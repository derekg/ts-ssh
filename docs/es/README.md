# ts-ssh: Herramienta CLI SSH/SCP Potente para Tailscale

Un cliente SSH de línea de comandos optimizado y utilidad SCP que se conecta a tu red Tailscale usando `tsnet`. Incluye operaciones multi-servidor potentes, ejecución de comandos por lotes, integración real con tmux, y una experiencia CLI moderna y hermosa - todo sin requerir el daemon completo de Tailscale.

Perfecto para equipos DevOps que necesitan acceso SSH rápido y confiable a través de su infraestructura Tailscale.

## Características

### 🚀 Funcionalidad Core SSH/SCP
*   **Conexión Tailscale en espacio de usuario** usando `tsnet` - no se requiere daemon
*   **Múltiples métodos de autenticación**: claves SSH, prompts de contraseña, o ambos
*   **Sesiones SSH interactivas** con soporte completo PTY y redimensionamiento de terminal
*   **Verificación segura de claves de servidor** usando `~/.ssh/known_hosts`
*   **Transferencias SCP directas** con detección automática de subida/descarga

### 💪 Operaciones Potentes Multi-Servidor
*   **`--list`**: Descubrimiento rápido de servidores con estado en línea/desconectado
*   **`--multi servidor1,servidor2,servidor3`**: Sesiones tmux reales con múltiples conexiones SSH
*   **`--exec "comando" servidor1,servidor2`**: Ejecución de comandos por lotes a través de servidores
*   **`--parallel`**: Ejecución concurrente de comandos para operaciones más rápidas
*   **`--copy archivo servidor1,servidor2:/ruta/`**: Distribución de archivos multi-servidor
*   **`--pick`**: Selección interactiva simple de servidores

### 🛠️ Características DevOps Profesionales
*   **Soporte ProxyCommand** (`-W`) para integración con herramientas estándar
*   **Multiplataforma**: Linux, macOS (Intel/ARM), Windows
*   **Experiencia CLI Moderna**: Estilo hermoso con framework Charmbracelet Fang
*   **Selección Interactiva de Servidores**: Selector mejorado con mejor UX
*   **Compatibilidad Legacy**: Compatibilidad completa hacia atrás para scripts existentes
*   **Inicio rápido** - sin frameworks de UI o inicialización compleja
*   **Comandos componibles** - funciona perfectamente en scripts y automatización
*   **Manejo claro de errores** y retroalimentación útil

## Modos CLI

ts-ssh soporta dos modos CLI para proporcionar tanto experiencia de usuario moderna como compatibilidad completa hacia atrás:

### 🎨 CLI Moderna (Predeterminada)
La experiencia CLI mejorada impulsada por el framework Fang de Charmbracelet proporciona:
- **Estilo hermoso** con colores consistentes y formato
- **Selección interactiva de servidores** con UX mejorada
- **Subcomandos estructurados** para funcionalidad organizada
- **Ayuda mejorada** con salida estilizada y mejor organización

```bash
# Ejemplos de uso CLI moderna
ts-ssh connect usuario@servidor           # Conexión SSH mejorada
ts-ssh list --verbose                     # Listado de servidores estilizado
ts-ssh multi web1,web2,db1               # Experiencia multi-servidor mejorada
ts-ssh copy archivo.txt servidor1,servidor2:/tmp/ # Operaciones de archivo mejoradas
```

### 🔧 CLI Legacy
Perfecto para scripts existentes y automatización que depende de la interfaz original:

```bash
# Forzar modo legacy con variable de entorno
export TS_SSH_LEGACY_CLI=1
ts-ssh --list                             # Comportamiento CLI original
ts-ssh usuario@servidor                   # Patrones de uso clásicos
```

**Detección Automática:**
- El modo legacy se activa automáticamente para patrones de uso amigables con scripts
- El modo moderno proporciona experiencia mejorada para uso interactivo
- Anular con variable de entorno `TS_SSH_LEGACY_CLI=1` cuando sea necesario

## Prerrequisitos

*   **Go:** Versión 1.18 o posterior instalada (`go version`).
*   **Cuenta Tailscale:** Una cuenta Tailscale activa.
*   **Nodo Destino:** Una máquina dentro de tu red Tailscale ejecutando un servidor SSH que permita conexiones desde tu usuario/clave/contraseña.

## Instalación

Puedes instalar `ts-ssh` usando `go install` (recomendado) o compilarlo manualmente desde el código fuente.

**Usando `go install`:**

```bash
go install github.com/derekg/ts-ssh@latest
```
*(Asegúrate de que tu `$GOPATH/bin` o `$HOME/go/bin` esté en el `PATH` de tu sistema)*

**Compilación Manual:**

1.  Clona el repositorio:
    ```bash
    git clone https://github.com/derekg/ts-ssh.git
    cd ts-ssh
    ```
2.  Compila el ejecutable:
    ```bash
    go build -o ts-ssh .
    ```
    Ahora puedes ejecutar `./ts-ssh`.

**Compilación Cruzada:**

Puedes compilar fácilmente para otras plataformas. Establece las variables de entorno `GOOS` y `GOARCH`. Usa `CGO_ENABLED=0` para una compilación cruzada más fácil.

*   **Para macOS (Apple Silicon):**
    ```bash
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ts-ssh-darwin-arm64 .
    ```
*   **Para macOS (Intel):**
    ```bash
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ts-ssh-darwin-amd64 .
    ```
*   **Para Linux (amd64):**
    ```bash
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ts-ssh-linux-amd64 .
    ```
*   **Para Windows (amd64):**
    ```bash
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ts-ssh-windows-amd64.exe .
    ```

## Uso

```
Uso: ts-ssh [opciones] [usuario@]servidor[:puerto] [comando...]
     ts-ssh --list                                    # Listar servidores disponibles
     ts-ssh --multi servidor1,servidor2,servidor3    # Sesión tmux multi-servidor
     ts-ssh --exec "comando" servidor1,servidor2     # Ejecutar comando en múltiples servidores
     ts-ssh --copy archivo.txt servidor1,servidor2:/tmp/ # Copiar archivo a múltiples servidores
     ts-ssh --pick                                   # Selector interactivo de servidores

Herramienta SSH/SCP potente para redes Tailscale.

Opciones:
  -W string
        reenviar stdio a servidor:puerto destino (para usar como ComandoProxy)
  -control-url string
        URL del plano de control Tailscale (opcional)
  -copy string
        Copiar archivos a múltiples servidores (formato: archivo_local servidor1,servidor2:/ruta/)
  -exec string
        Ejecutar comando en servidores especificados
  -i string
        Ruta a clave privada SSH (predeterminado "/home/user/.ssh/id_rsa")
  -insecure
        Deshabilitar verificación de clave de servidor (¡INSEGURO!)
  -l string
        Nombre de usuario SSH (predeterminado "user")
  -lang string
        Idioma para salida CLI (en, es)
  -list
        Listar servidores Tailscale disponibles
  -multi string
        Iniciar sesión tmux con múltiples servidores (separados por comas)
  -parallel
        Ejecutar comandos en paralelo (usar con --exec)
  -pick
        Selector interactivo de servidores (selección simple)
  -tsnet-dir string
        Directorio para almacenar estado tsnet (predeterminado "/home/user/.config/ts-ssh-client")
  -v    Logging detallado
  -version
        Mostrar versión y salir
```

**Argumentos:**

*   Para SSH: `[usuario@]servidor[:puerto] [comando...]`
    *   `servidor` **debe** ser el nombre MagicDNS de Tailscale o dirección IP Tailscale de la máquina destino.
    *   `usuario` predeterminado al nombre de usuario del SO actual si no se proporciona o especifica con `-l`.
    *   `puerto` predeterminado a `22` si no se proporciona.
    *   `comando...` (opcional): Si se proporciona, ejecuta el comando en el servidor remoto en lugar de iniciar un shell interactivo.
*   Para SCP (CLI directo):
    *   Subida: `ruta_local [usuario@]servidor:ruta_remota`
    *   Descarga: `[usuario@]servidor:ruta_remota ruta_local`
    *   El `usuario@` en el argumento remoto es opcional; si no se proporciona, se usará el usuario de `-l` o el usuario predeterminado del SO.

## Ejemplos

### 🔍 Descubrimiento de Servidores
```bash
# Listar todos los servidores Tailscale con estado
ts-ssh --list --lang es

# Información detallada de servidores
ts-ssh --list -v --lang es

# Selector interactivo de servidores
ts-ssh --pick --lang es
```

### 🖥️ Operaciones SSH Básicas
```bash
# Conectar a un solo servidor
ts-ssh tu-servidor

# Conectar como usuario específico
ts-ssh admin@tu-servidor
ts-ssh -l admin tu-servidor

# Ejecutar un comando remoto
ts-ssh tu-servidor uname -a

# Usar clave SSH específica
ts-ssh -i ~/.ssh/mi_clave usuario@tu-servidor

# Usar en español
TS_SSH_LANG=es ts-ssh tu-servidor
```

### 🚀 Operaciones Potentes Multi-Servidor
```bash
# Crear sesión tmux con múltiples servidores
ts-ssh --multi web1,web2,db1 --lang es

# Ejecutar comando en múltiples servidores (secuencial)
ts-ssh --exec "uptime" web1,web2,web3 --lang es

# Ejecutar comando en múltiples servidores (paralelo)
ts-ssh --parallel --exec "systemctl status nginx" web1,web2 --lang es

# Verificar espacio en disco en todos los servidores web
ts-ssh --exec "df -h" web1.dominio,web2.dominio,web3.dominio --lang es
```

### 📁 Operaciones de Transferencia de Archivos
```bash
# SCP a un solo servidor
ts-ssh local.txt tu-servidor:/ruta/remota/
ts-ssh tu-servidor:/archivo/remoto.txt ./

# Distribución de archivos multi-servidor
ts-ssh --copy deploy.sh web1,web2,web3:/tmp/ --lang es
ts-ssh --copy config.json db1,db2:/etc/miapp/ --lang es

# Copiar con usuario específico
ts-ssh --copy -l admin backup.tar.gz servidor1,servidor2:/backups/ --lang es
```

### 🔧 Uso Avanzado
```bash
# Integración ProxyCommand
scp -o ProxyCommand="ts-ssh -W %h:%p" archivo.txt servidor:/ruta/

# Información de versión
ts-ssh -version

# Logging detallado para depuración
ts-ssh --list -v --lang es

# Configurar idioma por defecto
export TS_SSH_LANG=es
ts-ssh --list
```

### 💡 Escenarios DevOps del Mundo Real
```bash
# Desplegar configuración a todos los servidores web
ts-ssh --copy nginx.conf web1,web2,web3:/etc/nginx/ --lang es
ts-ssh --parallel --exec "sudo nginx -t && sudo systemctl reload nginx" web1,web2,web3 --lang es

# Verificar estado del servicio a través de la infraestructura
ts-ssh --parallel --exec "systemctl is-active docker" nodo1,nodo2,nodo3 --lang es

# Recopilar logs de múltiples servidores
ts-ssh --exec "tail -100 /var/log/app.log" app1,app2,app3 --lang es

# Recopilación de información del sistema de emergencia
ts-ssh --parallel --exec "uptime && free -h && df -h" web1,web2,db1,db2 --lang es
```

## Configuración de Idioma

`ts-ssh` soporta múltiples idiomas para toda la salida CLI:

### Métodos de Configuración
1. **Bandera CLI**: `--lang es` (prioridad más alta)
2. **Variable de entorno**: `export TS_SSH_LANG=es`
3. **Predeterminado**: Inglés si no se especifica

### Idiomas Soportados
- **Inglés**: `en`, `english`, `en_us`, `en-us`
- **Español**: `es`, `spanish`, `español`, `es_es`, `es-es`, `es_mx`, `es-mx`
- **Chino**: `zh`, `chinese`, `中文`, `zh-cn`, `zh-tw`
- **Hindi**: `hi`, `hindi`, `हिन्दी`
- **Árabe**: `ar`, `arabic`, `العربية`
- **Bengalí**: `bn`, `bengali`, `বাংলা`
- **Portugués**: `pt`, `portuguese`, `português`, `pt-br`
- **Ruso**: `ru`, `russian`, `русский`
- **Japonés**: `ja`, `japanese`, `日本語`
- **Alemán**: `de`, `german`, `deutsch`
- **Francés**: `fr`, `french`, `français`

### Ejemplos de Configuración
```bash
# Usar español para un comando
ts-ssh --list --lang es

# Configurar español como predeterminado para la sesión
export TS_SSH_LANG=es
ts-ssh --list

# Agregar a tu perfil shell para configuración permanente
echo 'export TS_SSH_LANG=es' >> ~/.bashrc
```

## Sesiones tmux Multi-Servidor

La bandera `--multi` crea sesiones tmux reales con conexiones SSH a múltiples servidores. Esto proporciona una experiencia profesional de multiplexado de terminal:

```bash
# Crear sesión tmux con 3 servidores
ts-ssh --multi web1,web2,db1 --lang es
```

### Controles tmux
Una vez conectado, usa las combinaciones de teclas estándar de tmux:
- **`Ctrl+B n`** - Siguiente ventana (siguiente servidor)
- **`Ctrl+B p`** - Ventana anterior (servidor anterior)  
- **`Ctrl+B 1-9`** - Cambiar al número de ventana
- **`Ctrl+B c`** - Crear nueva ventana
- **`Ctrl+B d`** - Desconectar de la sesión
- **`Ctrl+B ?`** - Mostrar todas las combinaciones de teclas

### Gestión de Sesiones
```bash
# Listar sesiones tmux activas
tmux list-sessions

# Reconectar a una sesión desconectada
tmux attach-session -t ts-ssh-1234567890

# Eliminar una sesión específica
tmux kill-session -t ts-ssh-1234567890
```

**Nota:**
El flujo de autenticación de Tailscale puede mostrar logs detallados durante el inicio. Usa `-v` para una salida de diagnóstico más clara si es necesario.

## Autenticación Tailscale

La primera vez que ejecutas `ts-ssh` en una máquina, o si su autenticación Tailscale expira, necesitará autenticarse a tu red Tailscale.

El programa imprimirá una URL en la consola. Copia esta URL y ábrela en un navegador web. Inicia sesión en tu cuenta Tailscale para autorizar este cliente ("ts-ssh-client" o el hostname establecido en el código).

Una vez autorizado, `ts-ssh` almacena las claves de autenticación en el directorio de estado (`~/.config/ts-ssh-client` por defecto, configurable con `-tsnet-dir`) así que no necesitas re-autenticarte cada vez.

## Notas de Seguridad

*   **Verificación de Clave de Servidor:** Esta herramienta realiza verificación de clave de servidor contra `~/.ssh/known_hosts` por defecto. Esta es una característica de seguridad crucial para prevenir ataques Man-in-the-Middle (MITM).
*   **Bandera `-insecure`:** La bandera `-insecure` deshabilita la verificación de clave de servidor completamente. **Esto es peligroso** y solo debe usarse en entornos confiables o para propósitos específicos de prueba donde entiendes completamente las implicaciones de seguridad. Eres vulnerable a ataques MITM si usas esta bandera descuidadamente.

## Licencia

Este proyecto está licenciado bajo la Licencia MIT - consulta el archivo [LICENSE](../../LICENSE) para detalles.