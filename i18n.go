package main

import (
	"os"
	"strings"
	"sync"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Supported languages
const (
	LangEnglish = "en"
	LangSpanish = "es"
)

var (
	// Global printer for internationalization
	printer *message.Printer
	
	// Synchronization for thread-safe access
	initI18nOnce sync.Once
	printerMu    sync.RWMutex
	
	// Available languages
	supportedLanguages = map[string]language.Tag{
		LangEnglish: language.English,
		LangSpanish: language.Spanish,
	}
)

// initI18n initializes the internationalization system thread-safely
func initI18n(langFlag string) {
	// Ensure messages are registered only once across all goroutines
	initI18nOnce.Do(func() {
		registerMessages()
	})
	
	// Determine language preference: CLI flag > env var > default
	lang := determineLang(langFlag)
	
	// Get language tag
	tag, exists := supportedLanguages[lang]
	if !exists {
		tag = language.English // fallback to English
	}
	
	// Create printer for the selected language with thread-safe access
	printerMu.Lock()
	printer = message.NewPrinter(tag)
	printerMu.Unlock()
}

// determineLang determines which language to use based on priority:
// 1. CLI flag (--lang)
// 2. Environment variable (TS_SSH_LANG)
// 3. Standard locale environment variables (LC_ALL, LANG)
// 4. Default (English)
func determineLang(langFlag string) string {
	// Check CLI flag first
	if langFlag != "" {
		return normalizeLanguage(langFlag)
	}
	
	// Check custom environment variable
	if envLang := os.Getenv("TS_SSH_LANG"); envLang != "" {
		return normalizeLanguage(envLang)
	}
	
	// Check standard locale environment variables
	if envLang := os.Getenv("LC_ALL"); envLang != "" {
		return normalizeLanguage(envLang)
	}
	
	if envLang := os.Getenv("LANG"); envLang != "" {
		return normalizeLanguage(envLang)
	}
	
	// Default to English
	return LangEnglish
}

// normalizeLanguage normalizes language codes to our supported format
func normalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	
	// Handle common variations
	switch lang {
	case "en", "english", "en_us", "en-us":
		return LangEnglish
	case "es", "spanish", "español", "es_es", "es-es", "es_mx", "es-mx":
		return LangSpanish
	default:
		return LangEnglish // fallback
	}
}

// registerMessages registers all translatable messages
func registerMessages() {
	// Help and usage messages
	message.SetString(language.English, "usage_header", "Usage: %s [options] [user@]hostname[:port] [command...]")
	message.SetString(language.Spanish, "usage_header", "Uso: %s [opciones] [usuario@]servidor[:puerto] [comando...]")
	
	message.SetString(language.English, "usage_list", "       %s --list                                    # List available hosts")
	message.SetString(language.Spanish, "usage_list", "       %s --list                                    # Listar servidores disponibles")
	
	message.SetString(language.English, "usage_multi", "       %s --multi host1,host2,host3                # Multi-host tmux session")
	message.SetString(language.Spanish, "usage_multi", "       %s --multi servidor1,servidor2,servidor3    # Sesión tmux multi-servidor")
	
	message.SetString(language.English, "usage_exec", "       %s --exec \"command\" host1,host2             # Run command on multiple hosts")
	message.SetString(language.Spanish, "usage_exec", "       %s --exec \"comando\" servidor1,servidor2     # Ejecutar comando en múltiples servidores")
	
	message.SetString(language.English, "usage_copy", "       %s --copy file.txt host1,host2:/tmp/        # Copy file to multiple hosts")
	message.SetString(language.Spanish, "usage_copy", "       %s --copy archivo.txt servidor1,servidor2:/tmp/ # Copiar archivo a múltiples servidores")
	
	message.SetString(language.English, "usage_pick", "       %s --pick                                   # Interactive host picker")
	message.SetString(language.Spanish, "usage_pick", "       %s --pick                                   # Selector interactivo de servidores")
	
	message.SetString(language.English, "usage_description", "Powerful SSH/SCP tool for Tailscale networks.\n\nOptions:")
	message.SetString(language.Spanish, "usage_description", "Herramienta SSH/SCP potente para redes Tailscale.\n\nOpciones:")
	
	message.SetString(language.English, "examples_header", "\nExamples:")
	message.SetString(language.Spanish, "examples_header", "\nEjemplos:")
	
	message.SetString(language.English, "examples_basic_ssh", "  Basic SSH:")
	message.SetString(language.Spanish, "examples_basic_ssh", "  SSH básico:")
	
	message.SetString(language.English, "examples_interactive", "    %s user@host                    # Interactive SSH session")
	message.SetString(language.Spanish, "examples_interactive", "    %s usuario@servidor             # Sesión SSH interactiva")
	
	message.SetString(language.English, "examples_remote_cmd", "    %s user@host ls -lah            # Run remote command")
	message.SetString(language.Spanish, "examples_remote_cmd", "    %s usuario@servidor ls -lah     # Ejecutar comando remoto")
	
	message.SetString(language.English, "examples_host_discovery", "\n  Host Discovery:")
	message.SetString(language.Spanish, "examples_host_discovery", "\n  Descubrimiento de servidores:")
	
	message.SetString(language.English, "examples_list_hosts", "    %s --list                       # Show all Tailscale hosts")
	message.SetString(language.Spanish, "examples_list_hosts", "    %s --list                       # Mostrar todos los servidores Tailscale")
	
	message.SetString(language.English, "examples_pick_host", "    %s --pick                       # Pick host interactively")
	message.SetString(language.Spanish, "examples_pick_host", "    %s --pick                       # Elegir servidor interactivamente")
	
	message.SetString(language.English, "examples_multi_host", "\n  Multi-Host Operations:")
	message.SetString(language.Spanish, "examples_multi_host", "\n  Operaciones multi-servidor:")
	
	message.SetString(language.English, "examples_tmux", "    %s --multi web1,web2,db1        # Tmux session with 3 hosts")
	message.SetString(language.Spanish, "examples_tmux", "    %s --multi web1,web2,db1        # Sesión tmux con 3 servidores")
	
	message.SetString(language.English, "examples_exec_multi", "    %s --exec \"uptime\" web1,web2    # Run command on 2 hosts")
	message.SetString(language.Spanish, "examples_exec_multi", "    %s --exec \"uptime\" web1,web2    # Ejecutar comando en 2 servidores")
	
	message.SetString(language.English, "examples_parallel", "    %s --parallel --exec \"ps aux\" web1,web2  # Parallel execution")
	message.SetString(language.Spanish, "examples_parallel", "    %s --parallel --exec \"ps aux\" web1,web2  # Ejecución paralela")
	
	message.SetString(language.English, "examples_file_transfer", "\n  File Transfer:")
	message.SetString(language.Spanish, "examples_file_transfer", "\n  Transferencia de archivos:")
	
	message.SetString(language.English, "examples_scp_single", "    %s local.txt user@host:/remote/ # Single SCP upload")
	message.SetString(language.Spanish, "examples_scp_single", "    %s local.txt usuario@servidor:/remoto/ # Subida SCP única")
	
	message.SetString(language.English, "examples_scp_multi", "    %s --copy deploy.sh web1,web2:/tmp/  # Multi-host SCP")
	message.SetString(language.Spanish, "examples_scp_multi", "    %s --copy deploy.sh web1,web2:/tmp/  # SCP multi-servidor")
	
	message.SetString(language.English, "examples_proxy", "\n  ProxyCommand:")
	message.SetString(language.Spanish, "examples_proxy", "\n  ComandoProxy:")
	
	message.SetString(language.English, "examples_proxy_cmd", "    %s -W host:port                 # Proxy stdio via Tailscale")
	message.SetString(language.Spanish, "examples_proxy_cmd", "    %s -W servidor:puerto           # Proxy stdio vía Tailscale")
	
	// Error messages
	message.SetString(language.English, "error_init_tailscale", "Failed to initialize Tailscale connection: %v")
	message.SetString(language.Spanish, "error_init_tailscale", "Error al inicializar conexión Tailscale: %v")
	
	message.SetString(language.English, "error_scp_failed", "SCP operation failed: %v")
	message.SetString(language.Spanish, "error_scp_failed", "Operación SCP falló: %v")
	
	message.SetString(language.English, "scp_success", "SCP operation completed successfully.")
	message.SetString(language.Spanish, "scp_success", "Operación SCP completada exitosamente.")
	
	message.SetString(language.English, "error_parsing_target", "Error parsing target for SSH: %v")
	message.SetString(language.Spanish, "error_parsing_target", "Error analizando destino para SSH: %v")
	
	message.SetString(language.English, "error_init_ssh", "Failed to initialize Tailscale connection for SSH: %v")
	message.SetString(language.Spanish, "error_init_ssh", "Error al inicializar conexión Tailscale para SSH: %v")
	
	// Authentication messages
	message.SetString(language.English, "enter_password", "Enter password for %s@%s: ")
	message.SetString(language.Spanish, "enter_password", "Ingresa contraseña para %s@%s: ")
	
	message.SetString(language.English, "host_key_warning", "WARNING: Host key verification is disabled!")
	message.SetString(language.Spanish, "host_key_warning", "ADVERTENCIA: ¡Verificación de clave de servidor deshabilitada!")
	
	message.SetString(language.English, "using_key_auth", "Using public key authentication: %s")
	message.SetString(language.Spanish, "using_key_auth", "Usando autenticación de clave pública: %s")
	
	message.SetString(language.English, "key_auth_failed", "Failed to load private key: %v. Will attempt password auth.")
	message.SetString(language.Spanish, "key_auth_failed", "Error cargando clave privada: %v. Se intentará autenticación por contraseña.")
	
	// Connection messages
	message.SetString(language.English, "dial_via_tsnet", "Dialing %s via tsnet...")
	message.SetString(language.Spanish, "dial_via_tsnet", "Conectando a %s vía tsnet...")
	
	message.SetString(language.English, "dial_failed", "Failed to dial %s via tsnet (is Tailscale connection up and host reachable?): %v")
	message.SetString(language.Spanish, "dial_failed", "Error conectando a %s vía tsnet (¿está la conexión Tailscale activa y el servidor accesible?): %v")
	
	message.SetString(language.English, "ssh_connection_established", "SSH connection established.")
	message.SetString(language.Spanish, "ssh_connection_established", "Conexión SSH establecida.")
	
	message.SetString(language.English, "ssh_connection_failed", "Failed to establish SSH connection to %s: %v")
	message.SetString(language.Spanish, "ssh_connection_failed", "Error estableciendo conexión SSH a %s: %v")
	
	message.SetString(language.English, "ssh_auth_failed", "SSH Authentication failed for user %s: %v")
	message.SetString(language.Spanish, "ssh_auth_failed", "Autenticación SSH falló para usuario %s: %v")
	
	message.SetString(language.English, "host_key_failed", "SSH Host key verification failed: %v")
	message.SetString(language.Spanish, "host_key_failed", "Verificación de clave de servidor SSH falló: %v")
	
	message.SetString(language.English, "escape_sequence", "\nEscape sequence: ~. to terminate session")
	message.SetString(language.Spanish, "escape_sequence", "\nSecuencia de escape: ~. para terminar sesión")
	
	message.SetString(language.English, "ssh_session_closed", "SSH session closed.")
	message.SetString(language.Spanish, "ssh_session_closed", "Sesión SSH cerrada.")
	
	// Host list messages
	message.SetString(language.English, "no_peers_found", "No Tailscale peers found")
	message.SetString(language.Spanish, "no_peers_found", "No se encontraron pares Tailscale")
	
	message.SetString(language.English, "host_list_labels", "HOST,IP,STATUS,OS")
	message.SetString(language.Spanish, "host_list_labels", "SERVIDOR,IP,ESTADO,SO")
	
	message.SetString(language.English, "host_list_separator", "----,--,------,--")
	message.SetString(language.Spanish, "host_list_separator", "--------,--,------,--")
	
	message.SetString(language.English, "status_online", "ONLINE")
	message.SetString(language.Spanish, "status_online", "EN LÍNEA")
	
	message.SetString(language.English, "status_offline", "OFFLINE")
	message.SetString(language.Spanish, "status_offline", "DESCONECTADO")
	
	// Host picker messages
	message.SetString(language.English, "no_online_hosts", "no online hosts found")
	message.SetString(language.Spanish, "no_online_hosts", "no se encontraron servidores en línea")
	
	message.SetString(language.English, "available_hosts", "Available hosts:")
	message.SetString(language.Spanish, "available_hosts", "Servidores disponibles:")
	
	message.SetString(language.English, "select_host", "\nSelect host (1-%d): ")
	message.SetString(language.Spanish, "select_host", "\nSelecciona servidor (1-%d): ")
	
	message.SetString(language.English, "invalid_selection", "invalid selection")
	message.SetString(language.Spanish, "invalid_selection", "selección inválida")
	
	message.SetString(language.English, "selection_out_of_range", "selection out of range")
	message.SetString(language.Spanish, "selection_out_of_range", "selección fuera de rango")
	
	message.SetString(language.English, "connecting_to", "Connecting to %s...")
	message.SetString(language.Spanish, "connecting_to", "Conectando a %s...")
	
	// Multi-host operation messages
	message.SetString(language.English, "no_hosts_specified", "no hosts specified")
	message.SetString(language.Spanish, "no_hosts_specified", "no se especificaron servidores")
	
	message.SetString(language.English, "no_hosts_for_exec", "no hosts specified for --exec")
	message.SetString(language.Spanish, "no_hosts_for_exec", "no se especificaron servidores para --exec")
	
	message.SetString(language.English, "invalid_copy_format", "invalid --copy format. Use: localfile host1,host2:/path/")
	message.SetString(language.Spanish, "invalid_copy_format", "formato --copy inválido. Usar: archivo_local servidor1,servidor2:/ruta/")
	
	message.SetString(language.English, "invalid_remote_spec", "invalid remote specification. Must include path after ':'")
	message.SetString(language.Spanish, "invalid_remote_spec", "especificación remota inválida. Debe incluir ruta después de ':'")
	
	message.SetString(language.English, "copying_to", "Copying %s to %s:%s...")
	message.SetString(language.Spanish, "copying_to", "Copiando %s a %s:%s...")
	
	message.SetString(language.English, "copy_failed", "Failed to copy to %s: %v")
	message.SetString(language.Spanish, "copy_failed", "Error copiando a %s: %v")
	
	message.SetString(language.English, "copy_success", "Successfully copied to %s")
	message.SetString(language.Spanish, "copy_success", "Copiado exitosamente a %s")
	
	// SCP error messages
	message.SetString(language.English, "invalid_scp_remote", "invalid remote SCP argument format: %q. Must be [user@]host:path")
	message.SetString(language.Spanish, "invalid_scp_remote", "formato de argumento SCP remoto inválido: %q. Debe ser [usuario@]servidor:ruta")
	
	message.SetString(language.English, "invalid_user_host", "invalid user@host format in SCP argument: %q")
	message.SetString(language.Spanish, "invalid_user_host", "formato usuario@servidor inválido en argumento SCP: %q")
	
	message.SetString(language.English, "empty_host_scp", "host cannot be empty in SCP argument: %q")
	message.SetString(language.Spanish, "empty_host_scp", "el servidor no puede estar vacío en argumento SCP: %q")
	
	// Flag descriptions
	message.SetString(language.English, "flag_lang_desc", "Language for CLI output (en, es)")
	message.SetString(language.Spanish, "flag_lang_desc", "Idioma para salida CLI (en, es)")
	
	message.SetString(language.English, "flag_user_desc", "SSH Username")
	message.SetString(language.Spanish, "flag_user_desc", "Nombre de usuario SSH")
	
	message.SetString(language.English, "flag_key_desc", "Path to SSH private key")
	message.SetString(language.Spanish, "flag_key_desc", "Ruta a clave privada SSH")
	
	message.SetString(language.English, "flag_ssh_config_desc", "SSH configuration file")
	message.SetString(language.Spanish, "flag_ssh_config_desc", "Archivo de configuración SSH")
	
	message.SetString(language.English, "flag_tsnet_desc", "Directory to store tsnet state")
	message.SetString(language.Spanish, "flag_tsnet_desc", "Directorio para almacenar estado tsnet")
	
	message.SetString(language.English, "flag_control_desc", "Tailscale control plane URL (optional)")
	message.SetString(language.Spanish, "flag_control_desc", "URL del plano de control Tailscale (opcional)")
	
	message.SetString(language.English, "flag_verbose_desc", "Verbose logging")
	message.SetString(language.Spanish, "flag_verbose_desc", "Logging detallado")
	
	message.SetString(language.English, "flag_insecure_desc", "Disable host key checking (INSECURE!)")
	message.SetString(language.Spanish, "flag_insecure_desc", "Deshabilitar verificación de clave de servidor (¡INSEGURO!)")
	
	message.SetString(language.English, "flag_force_insecure_desc", "Skip confirmation for insecure connections (automation only)")
	message.SetString(language.Spanish, "flag_force_insecure_desc", "Omitir confirmación para conexiones inseguras (solo automatización)")
	
	message.SetString(language.English, "flag_forward_desc", "forward stdio to destination host:port (for use as ProxyCommand)")
	message.SetString(language.Spanish, "flag_forward_desc", "reenviar stdio a servidor:puerto destino (para usar como ComandoProxy)")
	
	message.SetString(language.English, "flag_version_desc", "Print version and exit")
	message.SetString(language.Spanish, "flag_version_desc", "Mostrar versión y salir")
	
	message.SetString(language.English, "flag_list_desc", "List available Tailscale hosts")
	message.SetString(language.Spanish, "flag_list_desc", "Listar servidores Tailscale disponibles")
	
	message.SetString(language.English, "flag_multi_desc", "Start tmux session with multiple hosts (comma-separated)")
	message.SetString(language.Spanish, "flag_multi_desc", "Iniciar sesión tmux con múltiples servidores (separados por comas)")
	
	message.SetString(language.English, "flag_exec_desc", "Execute command on specified hosts")
	message.SetString(language.Spanish, "flag_exec_desc", "Ejecutar comando en servidores especificados")
	
	message.SetString(language.English, "flag_copy_desc", "Copy files to multiple hosts (format: localfile host1,host2:/path/)")
	message.SetString(language.Spanish, "flag_copy_desc", "Copiar archivos a múltiples servidores (formato: archivo_local servidor1,servidor2:/ruta/)")
	
	message.SetString(language.English, "flag_pick_desc", "Interactive host picker (simple selection)")
	message.SetString(language.Spanish, "flag_pick_desc", "Selector interactivo de servidores (selección simple)")
	
	message.SetString(language.English, "flag_parallel_desc", "Execute commands in parallel (use with --exec)")
	message.SetString(language.Spanish, "flag_parallel_desc", "Ejecutar comandos en paralelo (usar con --exec)")
	
	// SCP-specific messages
	message.SetString(language.English, "scp_enter_password", "Enter password for %s@%s (for SCP): ")
	message.SetString(language.Spanish, "scp_enter_password", "Ingresa contraseña para %s@%s (para SCP): ")
	
	message.SetString(language.English, "scp_host_key_warning", "CLI SCP: WARNING! Host key verification is disabled!")
	message.SetString(language.Spanish, "scp_host_key_warning", "CLI SCP: ¡ADVERTENCIA! ¡Verificación de clave de servidor deshabilitada!")
	
	message.SetString(language.English, "scp_empty_path", "local or remote path for SCP cannot be empty")
	message.SetString(language.Spanish, "scp_empty_path", "la ruta local o remota para SCP no puede estar vacía")
	
	message.SetString(language.English, "scp_upload_complete", "CLI SCP: Upload complete.")
	message.SetString(language.Spanish, "scp_upload_complete", "CLI SCP: Subida completada.")
	
	message.SetString(language.English, "scp_download_complete", "CLI SCP: Download complete.")
	message.SetString(language.Spanish, "scp_download_complete", "CLI SCP: Descarga completada.")
	
	// Common error messages
	message.SetString(language.English, "error_prefix", "Error: %v")
	message.SetString(language.Spanish, "error_prefix", "Error: %v")
	
	message.SetString(language.English, "failed_read_user_input", "failed to read user input: %w")
	message.SetString(language.Spanish, "failed_read_user_input", "error al leer entrada del usuario: %w")
	
	message.SetString(language.English, "hostname_cannot_be_empty", "hostname cannot be empty")
	message.SetString(language.Spanish, "hostname_cannot_be_empty", "el nombre del servidor no puede estar vacío")
	
	message.SetString(language.English, "invalid_port_number", "invalid port number '%s': %w")
	message.SetString(language.Spanish, "invalid_port_number", "número de puerto inválido '%s': %w")
	
	message.SetString(language.English, "invalid_host_port_format", "invalid host:port format '%s': %w")
	message.SetString(language.Spanish, "invalid_host_port_format", "formato servidor:puerto inválido '%s': %w")
	
	// TTY and security messages
	message.SetString(language.English, "not_running_in_terminal", "not running in a terminal")
	message.SetString(language.Spanish, "not_running_in_terminal", "no se está ejecutando en una terminal")
	
	message.SetString(language.English, "tty_security_validation_failed", "TTY security validation failed: %w")
	message.SetString(language.Spanish, "tty_security_validation_failed", "falló la validación de seguridad TTY: %w")
	
	message.SetString(language.English, "failed_open_tty", "failed to open TTY: %w")
	message.SetString(language.Spanish, "failed_open_tty", "error al abrir TTY: %w")
	
	// Security warning messages for insecure mode
	message.SetString(language.English, "warning_insecure_mode", "Host key verification disabled!")
	message.SetString(language.Spanish, "warning_insecure_mode", "¡Verificación de clave de servidor deshabilitada!")
	
	message.SetString(language.English, "warning_mitm_vulnerability", "This makes you vulnerable to man-in-the-middle attacks.")
	message.SetString(language.Spanish, "warning_mitm_vulnerability", "Esto te hace vulnerable a ataques de intermediario (man-in-the-middle).")
	
	message.SetString(language.English, "warning_trusted_networks_only", "Only use this in trusted network environments.")
	message.SetString(language.Spanish, "warning_trusted_networks_only", "Solo usa esto en entornos de red confiables.")
	
	message.SetString(language.English, "insecure_mode_forced", "Insecure mode forced via --force-insecure flag.")
	message.SetString(language.Spanish, "insecure_mode_forced", "Modo inseguro forzado mediante flag --force-insecure.")
	
	message.SetString(language.English, "confirm_insecure_connection", "Continue with insecure connection? [y/N]:")
	message.SetString(language.Spanish, "confirm_insecure_connection", "¿Continuar con conexión insegura? [y/N]:")
	
	message.SetString(language.English, "connection_cancelled_by_user", "connection cancelled by user")
	message.SetString(language.Spanish, "connection_cancelled_by_user", "conexión cancelada por el usuario")
	
	message.SetString(language.English, "proceeding_with_insecure_connection", "Proceeding with insecure connection...")
	message.SetString(language.Spanish, "proceeding_with_insecure_connection", "Procediendo con conexión insegura...")
	
	// CLI command descriptions for fang
	message.SetString(language.English, "cli_description", "Secure SSH/SCP client with Tailscale connectivity for enterprise environments")
	message.SetString(language.Spanish, "cli_description", "Cliente SSH/SCP seguro con conectividad Tailscale para entornos empresariales")
	
	message.SetString(language.English, "cmd_connect_desc", "Connect to a remote host via SSH (default command)")
	message.SetString(language.Spanish, "cmd_connect_desc", "Conectar a un servidor remoto via SSH (comando por defecto)")
	
	message.SetString(language.English, "cmd_scp_desc", "Transfer files securely using SCP")
	message.SetString(language.Spanish, "cmd_scp_desc", "Transferir archivos de forma segura usando SCP")
	
	message.SetString(language.English, "cmd_list_desc", "List available Tailscale hosts")
	message.SetString(language.Spanish, "cmd_list_desc", "Listar servidores Tailscale disponibles")
	
	message.SetString(language.English, "cmd_exec_desc", "Execute commands on multiple hosts")
	message.SetString(language.Spanish, "cmd_exec_desc", "Ejecutar comandos en múltiples servidores")
	
	message.SetString(language.English, "cmd_multi_desc", "Multi-host operations with tmux session management")
	message.SetString(language.Spanish, "cmd_multi_desc", "Operaciones multi-servidor con gestión de sesiones tmux")
	
	message.SetString(language.English, "cmd_config_desc", "Manage application configuration")
	message.SetString(language.Spanish, "cmd_config_desc", "Gestionar configuración de la aplicación")
	
	message.SetString(language.English, "cmd_pqc_desc", "Post-quantum cryptography operations and reporting")
	message.SetString(language.Spanish, "cmd_pqc_desc", "Operaciones y reportes de criptografía post-cuántica")
	
	message.SetString(language.English, "cmd_version_desc", "Show version information")
	message.SetString(language.Spanish, "cmd_version_desc", "Mostrar información de versión")
}

// T returns a localized string using the global printer thread-safely
func T(key string, args ...interface{}) string {
	// Read printer with read lock for concurrent access
	printerMu.RLock()
	p := printer
	printerMu.RUnlock()
	
	// Initialize if not yet done
	if p == nil {
		initI18n("")
		printerMu.RLock()
		p = printer
		printerMu.RUnlock()
	}
	
	// Use local copy to avoid holding lock during sprintf
	return p.Sprintf(key, args...)
}