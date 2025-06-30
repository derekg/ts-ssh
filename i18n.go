package main

import (
	"os"
	"strings"
	"sync"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Supported languages (Top 10 most popular languages by speakers)
const (
	LangEnglish    = "en"
	LangSpanish    = "es"
	LangChinese    = "zh"
	LangHindi      = "hi"
	LangArabic     = "ar"
	LangBengali    = "bn"
	LangPortuguese = "pt"
	LangRussian    = "ru"
	LangJapanese   = "ja"
	LangGerman     = "de"
	LangFrench     = "fr"
)

var (
	// Global printer for internationalization
	printer *message.Printer
	
	// Synchronization for thread-safe access
	initI18nOnce sync.Once
	printerMu    sync.RWMutex
	
	// Available languages
	supportedLanguages = map[string]language.Tag{
		LangEnglish:    language.English,
		LangSpanish:    language.Spanish,
		LangChinese:    language.Chinese,
		LangHindi:      language.Hindi,
		LangArabic:     language.Arabic,
		LangBengali:    language.Bengali,
		LangPortuguese: language.Portuguese,
		LangRussian:    language.Russian,
		LangJapanese:   language.Japanese,
		LangGerman:     language.German,
		LangFrench:     language.French,
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
	switch {
	case strings.HasPrefix(lang, "en") || lang == "english":
		return LangEnglish
	case strings.HasPrefix(lang, "es") || lang == "spanish" || lang == "español":
		return LangSpanish
	case strings.HasPrefix(lang, "zh") || lang == "chinese" || lang == "中文":
		return LangChinese
	case strings.HasPrefix(lang, "hi") || lang == "hindi" || lang == "हिन्दी":
		return LangHindi
	case strings.HasPrefix(lang, "ar") || lang == "arabic" || lang == "العربية":
		return LangArabic
	case strings.HasPrefix(lang, "bn") || lang == "bengali" || lang == "বাংলা":
		return LangBengali
	case strings.HasPrefix(lang, "pt") || lang == "portuguese" || lang == "português":
		return LangPortuguese
	case strings.HasPrefix(lang, "ru") || lang == "russian" || lang == "русский":
		return LangRussian
	case strings.HasPrefix(lang, "ja") || lang == "japanese" || lang == "日本語":
		return LangJapanese
	case strings.HasPrefix(lang, "de") || lang == "german" || lang == "deutsch":
		return LangGerman
	case strings.HasPrefix(lang, "fr") || lang == "french" || lang == "français":
		return LangFrench
	default:
		return LangEnglish // fallback
	}
}

// registerMessages registers all translatable messages
func registerMessages() {
	// Help and usage messages
	message.SetString(language.English, "usage_header", "Usage: %s [options] [user@]hostname[:port] [command...]")
	message.SetString(language.Spanish, "usage_header", "Uso: %s [opciones] [usuario@]servidor[:puerto] [comando...]")
	message.SetString(language.Chinese, "usage_header", "用法: %s [选项] [用户@]主机名[:端口] [命令...]")
	message.SetString(language.Hindi, "usage_header", "उपयोग: %s [विकल्प] [उपयोगकर्ता@]होस्टनाम[:पोर्ट] [कमांड...]")
	message.SetString(language.Arabic, "usage_header", "الاستخدام: %s [خيارات] [مستخدم@]اسم_المضيف[:منفذ] [أمر...]")
	message.SetString(language.Bengali, "usage_header", "ব্যবহার: %s [বিকল্প] [ব্যবহারকারী@]হোস্টনাম[:পোর্ট] [কমান্ড...]")
	message.SetString(language.Portuguese, "usage_header", "Uso: %s [opções] [usuário@]hostname[:porta] [comando...]")
	message.SetString(language.Russian, "usage_header", "Использование: %s [опции] [пользователь@]хост[:порт] [команда...]")
	message.SetString(language.Japanese, "usage_header", "使用法: %s [オプション] [ユーザー@]ホスト名[:ポート] [コマンド...]")
	message.SetString(language.German, "usage_header", "Verwendung: %s [Optionen] [Benutzer@]Hostname[:Port] [Befehl...]")
	message.SetString(language.French, "usage_header", "Utilisation: %s [options] [utilisateur@]nom_hôte[:port] [commande...]")
	
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
	message.SetString(language.Chinese, "error_init_tailscale", "初始化 Tailscale 连接失败: %v")
	message.SetString(language.Hindi, "error_init_tailscale", "Tailscale कनेक्शन प्रारंभ करने में विफल: %v")
	message.SetString(language.Arabic, "error_init_tailscale", "فشل في تهيئة اتصال Tailscale: %v")
	message.SetString(language.Bengali, "error_init_tailscale", "Tailscale সংযোগ শুরু করতে ব্যর্থ: %v")
	message.SetString(language.Portuguese, "error_init_tailscale", "Falha ao inicializar conexão Tailscale: %v")
	message.SetString(language.Russian, "error_init_tailscale", "Не удалось инициализировать соединение Tailscale: %v")
	message.SetString(language.Japanese, "error_init_tailscale", "Tailscale接続の初期化に失敗しました: %v")
	message.SetString(language.German, "error_init_tailscale", "Fehler beim Initialisieren der Tailscale-Verbindung: %v")
	message.SetString(language.French, "error_init_tailscale", "Échec de l'initialisation de la connexion Tailscale: %v")
	
	message.SetString(language.English, "error_scp_failed", "SCP operation failed: %v")
	message.SetString(language.Spanish, "error_scp_failed", "Operación SCP falló: %v")
	message.SetString(language.Chinese, "error_scp_failed", "SCP 操作失败: %v")
	message.SetString(language.Hindi, "error_scp_failed", "SCP ऑपरेशन विफल: %v")
	message.SetString(language.Arabic, "error_scp_failed", "فشلت عملية SCP: %v")
	message.SetString(language.Bengali, "error_scp_failed", "SCP অপারেশন ব্যর্থ: %v")
	message.SetString(language.Portuguese, "error_scp_failed", "Operação SCP falhou: %v")
	message.SetString(language.Russian, "error_scp_failed", "Операция SCP не удалась: %v")
	message.SetString(language.Japanese, "error_scp_failed", "SCP操作が失敗しました: %v")
	message.SetString(language.German, "error_scp_failed", "SCP-Operation fehlgeschlagen: %v")
	message.SetString(language.French, "error_scp_failed", "L'opération SCP a échoué: %v")
	
	message.SetString(language.English, "scp_success", "SCP operation completed successfully.")
	message.SetString(language.Spanish, "scp_success", "Operación SCP completada exitosamente.")
	message.SetString(language.Chinese, "scp_success", "SCP 操作成功完成。")
	message.SetString(language.Hindi, "scp_success", "SCP ऑपरेशन सफलतापूर्वक पूरा हुआ।")
	message.SetString(language.Arabic, "scp_success", "تمت عملية SCP بنجاحق")
	message.SetString(language.Bengali, "scp_success", "SCP অপারেশন সফলভাবে সম্পন্ন হয়েছে।")
	message.SetString(language.Portuguese, "scp_success", "Operação SCP concluída com sucesso.")
	message.SetString(language.Russian, "scp_success", "Операция SCP успешно завершена.")
	message.SetString(language.Japanese, "scp_success", "SCP操作が正常に完了しました。")
	message.SetString(language.German, "scp_success", "SCP-Operation erfolgreich abgeschlossen.")
	message.SetString(language.French, "scp_success", "Opération SCP terminée avec succès.")
	
	message.SetString(language.English, "error_parsing_target", "Error parsing target for SSH: %v")
	message.SetString(language.Spanish, "error_parsing_target", "Error analizando destino para SSH: %v")
	message.SetString(language.Chinese, "error_parsing_target", "解析 SSH 目标错误: %v")
	message.SetString(language.Hindi, "error_parsing_target", "SSH के लिए लक्ष्य पार्स करने में त्रुटि: %v")
	message.SetString(language.Arabic, "error_parsing_target", "خطأ في تحليل الهدف لـ SSH: %v")
	message.SetString(language.Bengali, "error_parsing_target", "SSH এর জন্য টার্গেট পার্স করার ত্রুটি: %v")
	message.SetString(language.Portuguese, "error_parsing_target", "Erro ao analisar destino para SSH: %v")
	message.SetString(language.Russian, "error_parsing_target", "Ошибка разбора цели для SSH: %v")
	message.SetString(language.Japanese, "error_parsing_target", "SSH のターゲット解析エラー: %v")
	message.SetString(language.German, "error_parsing_target", "Fehler beim Parsen des SSH-Ziels: %v")
	message.SetString(language.French, "error_parsing_target", "Erreur lors de l'analyse de la cible SSH: %v")
	
	message.SetString(language.English, "error_init_ssh", "Failed to initialize Tailscale connection for SSH: %v")
	message.SetString(language.Spanish, "error_init_ssh", "Error al inicializar conexión Tailscale para SSH: %v")
	
	// Authentication messages
	message.SetString(language.English, "enter_password", "Enter password for %s@%s: ")
	message.SetString(language.Spanish, "enter_password", "Ingresa contraseña para %s@%s: ")
	message.SetString(language.Chinese, "enter_password", "输入 %s@%s 的密码: ")
	message.SetString(language.Hindi, "enter_password", "%s@%s के लिए पासवर्ड दर्ज करें: ")
	message.SetString(language.Arabic, "enter_password", "أدخل كلمة المرور لـ %s@%s: ")
	message.SetString(language.Bengali, "enter_password", "%s@%s এর জন্য পাসওয়ার্ড লিখুন: ")
	message.SetString(language.Portuguese, "enter_password", "Digite a senha para %s@%s: ")
	message.SetString(language.Russian, "enter_password", "Введите пароль для %s@%s: ")
	message.SetString(language.Japanese, "enter_password", "%s@%s のパスワードを入力: ")
	message.SetString(language.German, "enter_password", "Passwort für %s@%s eingeben: ")
	message.SetString(language.French, "enter_password", "Entrez le mot de passe pour %s@%s: ")
	
	message.SetString(language.English, "host_key_warning", "WARNING: Host key verification is disabled!")
	message.SetString(language.Spanish, "host_key_warning", "ADVERTENCIA: ¡Verificación de clave de servidor deshabilitada!")
	message.SetString(language.Chinese, "host_key_warning", "警告：主机密钥验证已禁用！")
	message.SetString(language.Hindi, "host_key_warning", "चेतावनी: होस्ट की सत्यापन अक्षम है!")
	message.SetString(language.Arabic, "host_key_warning", "تحذير: تم تعطيل التحقق من مفتاح المضيف!")
	message.SetString(language.Bengali, "host_key_warning", "সতর্কবার্তা: হোস্ট কী যাচাইকরণ নিষ্ক্রিয়!")
	message.SetString(language.Portuguese, "host_key_warning", "AVISO: Verificação de chave do host está desabilitada!")
	message.SetString(language.Russian, "host_key_warning", "ПРЕДУПРЕЖДЕНИЕ: Проверка ключа хоста отключена!")
	message.SetString(language.Japanese, "host_key_warning", "警告: ホストキーの検証が無効です!")
	message.SetString(language.German, "host_key_warning", "WARNUNG: Host-Schlüssel-Verifikation ist deaktiviert!")
	message.SetString(language.French, "host_key_warning", "AVERTISSEMENT: La vérification de la clé d'hôte est désactivée!")
	
	message.SetString(language.English, "using_key_auth", "Using public key authentication: %s")
	message.SetString(language.Spanish, "using_key_auth", "Usando autenticación de clave pública: %s")
	message.SetString(language.Chinese, "using_key_auth", "使用公钥认证: %s")
	message.SetString(language.Hindi, "using_key_auth", "पब्लिक की ऑथेंटिकेशन का उपयोग: %s")
	message.SetString(language.Arabic, "using_key_auth", "استخدام مصادقة المفتاح العام: %s")
	message.SetString(language.Bengali, "using_key_auth", "পাবলিক কী প্রমাণীকরণ ব্যবহার করা হচ্ছে: %s")
	message.SetString(language.Portuguese, "using_key_auth", "Usando autenticação por chave pública: %s")
	message.SetString(language.Russian, "using_key_auth", "Используется аутентификация по открытому ключу: %s")
	message.SetString(language.Japanese, "using_key_auth", "公開鍵認証を使用: %s")
	message.SetString(language.German, "using_key_auth", "Verwende öffentliche Schlüssel-Authentifizierung: %s")
	message.SetString(language.French, "using_key_auth", "Utilisation de l'authentification par clé publique: %s")
	
	message.SetString(language.English, "key_auth_failed", "Failed to load private key: %v. Will attempt password auth.")
	message.SetString(language.Spanish, "key_auth_failed", "Error cargando clave privada: %v. Se intentará autenticación por contraseña.")
	
	// Connection messages
	message.SetString(language.English, "dial_via_tsnet", "Dialing %s via tsnet...")
	message.SetString(language.Spanish, "dial_via_tsnet", "Conectando a %s vía tsnet...")
	
	message.SetString(language.English, "dial_failed", "Failed to dial %s via tsnet (is Tailscale connection up and host reachable?): %v")
	message.SetString(language.Spanish, "dial_failed", "Error conectando a %s vía tsnet (¿está la conexión Tailscale activa y el servidor accesible?): %v")
	message.SetString(language.Chinese, "dial_failed", "通过 tsnet 连接到 %s 失败（Tailscale 连接是否正常，主机是否可达？）: %v")
	message.SetString(language.Hindi, "dial_failed", "tsnet के माध्यम से %s को डायल करने में विफलता (Tailscale कनेक्शन चालू है और होस्ट पहुंचने योग्य है?): %v")
	message.SetString(language.Arabic, "dial_failed", "فشل في الاتصال بـ %s عبر tsnet (هل اتصال Tailscale نشط والمضيف قابل للوصول؟): %v")
	message.SetString(language.Bengali, "dial_failed", "tsnet এর মাধ্যমে %s এ ডায়েল করতে ব্যর্থ (Tailscale কনেকশন টিক আছে এবং হোস্ট পৌঁছানো যায়?): %v")
	message.SetString(language.Portuguese, "dial_failed", "Falha ao conectar com %s via tsnet (conexão Tailscale está ativa e host é alcançável?): %v")
	message.SetString(language.Russian, "dial_failed", "Не удалось соединиться с %s через tsnet (работает ли соединение Tailscale и доступен ли хост?): %v")
	message.SetString(language.Japanese, "dial_failed", "tsnet経由で%sへの接続に失敗 (Tailscale接続が有効でホストに到達可能ですか?): %v")
	message.SetString(language.German, "dial_failed", "Verbindung zu %s über tsnet fehlgeschlagen (ist Tailscale-Verbindung aktiv und Host erreichbar?): %v")
	message.SetString(language.French, "dial_failed", "Échec de la connexion à %s via tsnet (la connexion Tailscale est-elle active et l'hôte accessible?): %v")
	
	message.SetString(language.English, "ssh_connection_established", "SSH connection established.")
	message.SetString(language.Spanish, "ssh_connection_established", "Conexión SSH establecida.")
	message.SetString(language.Chinese, "ssh_connection_established", "SSH 连接已建立。")
	message.SetString(language.Hindi, "ssh_connection_established", "SSH कनेक्शन स्थापित हुआ।")
	message.SetString(language.Arabic, "ssh_connection_established", "تم إنشاء اتصال SSHآ")
	message.SetString(language.Bengali, "ssh_connection_established", "SSH কনেকশন স্থাপন করা হয়েছে।")
	message.SetString(language.Portuguese, "ssh_connection_established", "Conexão SSH estabelecida.")
	message.SetString(language.Russian, "ssh_connection_established", "SSH-соединение установлено.")
	message.SetString(language.Japanese, "ssh_connection_established", "SSH接続が確立されました。")
	message.SetString(language.German, "ssh_connection_established", "SSH-Verbindung hergestellt.")
	message.SetString(language.French, "ssh_connection_established", "Connexion SSH établie.")
	
	message.SetString(language.English, "ssh_connection_failed", "Failed to establish SSH connection to %s: %v")
	message.SetString(language.Spanish, "ssh_connection_failed", "Error estableciendo conexión SSH a %s: %v")
	message.SetString(language.Chinese, "ssh_connection_failed", "建立到 %s 的 SSH 连接失败: %v")
	message.SetString(language.Hindi, "ssh_connection_failed", "%s से SSH कनेक्शन स्थापित करने में विफल: %v")
	message.SetString(language.Arabic, "ssh_connection_failed", "فشل في إنشاء اتصال SSH إلى %s: %v")
	message.SetString(language.Bengali, "ssh_connection_failed", "%s এ SSH কনেকশন স্থাপন করতে ব্যর্থ: %v")
	message.SetString(language.Portuguese, "ssh_connection_failed", "Falha ao estabelecer conexão SSH para %s: %v")
	message.SetString(language.Russian, "ssh_connection_failed", "Не удалось установить SSH-соединение с %s: %v")
	message.SetString(language.Japanese, "ssh_connection_failed", "%s へのSSH接続の確立に失敗: %v")
	message.SetString(language.German, "ssh_connection_failed", "SSH-Verbindung zu %s fehlgeschlagen: %v")
	message.SetString(language.French, "ssh_connection_failed", "Échec de l'établissement de la connexion SSH vers %s: %v")
	
	message.SetString(language.English, "ssh_auth_failed", "SSH Authentication failed for user %s: %v")
	message.SetString(language.Spanish, "ssh_auth_failed", "Autenticación SSH falló para usuario %s: %v")
	
	message.SetString(language.English, "host_key_failed", "SSH Host key verification failed: %v")
	message.SetString(language.Spanish, "host_key_failed", "Verificación de clave de servidor SSH falló: %v")
	
	message.SetString(language.English, "escape_sequence", "\nEscape sequence: ~. to terminate session")
	message.SetString(language.Spanish, "escape_sequence", "\nSecuencia de escape: ~. para terminar sesión")
	message.SetString(language.Chinese, "escape_sequence", "\n退出序列: ~. 终止会话")
	message.SetString(language.Hindi, "escape_sequence", "\nएस्केप सीक्वेंस: ~. सत्र समाप्त करने के लिए")
	message.SetString(language.Arabic, "escape_sequence", "\nتسلسل الخروج: ~. لإنهاء الجلسة")
	message.SetString(language.Bengali, "escape_sequence", "\nএসকেপ সিকোয়েন্স: ~. সেশন সমাপ্ত করতে")
	message.SetString(language.Portuguese, "escape_sequence", "\nSequência de escape: ~. para terminar sessão")
	message.SetString(language.Russian, "escape_sequence", "\nПоследовательность выхода: ~. для завершения сессии")
	message.SetString(language.Japanese, "escape_sequence", "\nエスケープシーケンス: ~. セッションを終了")
	message.SetString(language.German, "escape_sequence", "\nEscape-Sequenz: ~. zum Beenden der Sitzung")
	message.SetString(language.French, "escape_sequence", "\nSéquence d'échappement: ~. pour terminer la session")
	
	message.SetString(language.English, "ssh_session_closed", "SSH session closed.")
	message.SetString(language.Spanish, "ssh_session_closed", "Sesión SSH cerrada.")
	
	// Host list messages
	message.SetString(language.English, "no_peers_found", "No Tailscale peers found")
	message.SetString(language.Spanish, "no_peers_found", "No se encontraron pares Tailscale")
	message.SetString(language.Chinese, "no_peers_found", "未找到 Tailscale 对等节点")
	message.SetString(language.Hindi, "no_peers_found", "कोई Tailscale पीयर नहीं मिला")
	message.SetString(language.Arabic, "no_peers_found", "لم يتم العثور على أقران Tailscale")
	message.SetString(language.Bengali, "no_peers_found", "কোন Tailscale পিয়ার পাওয়া যায়নি")
	message.SetString(language.Portuguese, "no_peers_found", "Nenhum par Tailscale encontrado")
	message.SetString(language.Russian, "no_peers_found", "Узлы Tailscale не найдены")
	message.SetString(language.Japanese, "no_peers_found", "Tailscaleピアが見つかりません")
	message.SetString(language.German, "no_peers_found", "Keine Tailscale-Peers gefunden")
	message.SetString(language.French, "no_peers_found", "Aucun pair Tailscale trouvé")
	
	message.SetString(language.English, "host_list_labels", "HOST,IP,STATUS,OS")
	message.SetString(language.Spanish, "host_list_labels", "SERVIDOR,IP,ESTADO,SO")
	
	message.SetString(language.English, "host_list_separator", "----,--,------,--")
	message.SetString(language.Spanish, "host_list_separator", "--------,--,------,--")
	
	message.SetString(language.English, "status_online", "ONLINE")
	message.SetString(language.Spanish, "status_online", "EN LÍNEA")
	message.SetString(language.Chinese, "status_online", "在线")
	message.SetString(language.Hindi, "status_online", "ऑनलाइन")
	message.SetString(language.Arabic, "status_online", "متصل")
	message.SetString(language.Bengali, "status_online", "অনলাইন")
	message.SetString(language.Portuguese, "status_online", "ONLINE")
	message.SetString(language.Russian, "status_online", "В СЕТИ")
	message.SetString(language.Japanese, "status_online", "オンライン")
	message.SetString(language.German, "status_online", "ONLINE")
	message.SetString(language.French, "status_online", "EN LIGNE")
	
	message.SetString(language.English, "status_offline", "OFFLINE")
	message.SetString(language.Spanish, "status_offline", "DESCONECTADO")
	message.SetString(language.Chinese, "status_offline", "离线")
	message.SetString(language.Hindi, "status_offline", "ऑफ़लाइन")
	message.SetString(language.Arabic, "status_offline", "غير متصل")
	message.SetString(language.Bengali, "status_offline", "অফলাইন")
	message.SetString(language.Portuguese, "status_offline", "OFFLINE")
	message.SetString(language.Russian, "status_offline", "НЕ В СЕТИ")
	message.SetString(language.Japanese, "status_offline", "オフライン")
	message.SetString(language.German, "status_offline", "OFFLINE")
	message.SetString(language.French, "status_offline", "HORS LIGNE")
	
	// Host picker messages
	message.SetString(language.English, "no_online_hosts", "no online hosts found")
	message.SetString(language.Spanish, "no_online_hosts", "no se encontraron servidores en línea")
	message.SetString(language.Chinese, "no_online_hosts", "未找到在线主机")
	message.SetString(language.Hindi, "no_online_hosts", "कोई ऑनलाइन होस्ट नहीं मिला")
	message.SetString(language.Arabic, "no_online_hosts", "لم يتم العثور على مضيفين متصلين")
	message.SetString(language.Bengali, "no_online_hosts", "কোন অনলাইন হোস्ট পাওয়া যায়নি")
	message.SetString(language.Portuguese, "no_online_hosts", "nenhum host online encontrado")
	message.SetString(language.Russian, "no_online_hosts", "онлайн хосты не найдены")
	message.SetString(language.Japanese, "no_online_hosts", "オンラインのホストが見つかりません")
	message.SetString(language.German, "no_online_hosts", "keine Online-Hosts gefunden")
	message.SetString(language.French, "no_online_hosts", "aucun hôte en ligne trouvé")
	
	message.SetString(language.English, "available_hosts", "Available hosts:")
	message.SetString(language.Spanish, "available_hosts", "Servidores disponibles:")
	message.SetString(language.Chinese, "available_hosts", "可用主机:")
	message.SetString(language.Hindi, "available_hosts", "उपलब्ध होस्ट:")
	message.SetString(language.Arabic, "available_hosts", "المضيفين المتاحين:")
	message.SetString(language.Bengali, "available_hosts", "উপলব্ধ হোস্ট:")
	message.SetString(language.Portuguese, "available_hosts", "Hosts disponíveis:")
	message.SetString(language.Russian, "available_hosts", "Доступные хосты:")
	message.SetString(language.Japanese, "available_hosts", "利用可能なホスト:")
	message.SetString(language.German, "available_hosts", "Verfügbare Hosts:")
	message.SetString(language.French, "available_hosts", "Hôtes disponibles:")
	
	message.SetString(language.English, "select_host", "\nSelect host (1-%d): ")
	message.SetString(language.Spanish, "select_host", "\nSelecciona servidor (1-%d): ")
	message.SetString(language.Chinese, "select_host", "\n选择主机 (1-%d): ")
	message.SetString(language.Hindi, "select_host", "\nहोस्ट चुनें (1-%d): ")
	message.SetString(language.Arabic, "select_host", "\nاختر المضيف (1-%d): ")
	message.SetString(language.Bengali, "select_host", "\nহোস্ট নির্বাচন করুন (1-%d): ")
	message.SetString(language.Portuguese, "select_host", "\nSelecionar host (1-%d): ")
	message.SetString(language.Russian, "select_host", "\nВыберите хост (1-%d): ")
	message.SetString(language.Japanese, "select_host", "\nホストを選択 (1-%d): ")
	message.SetString(language.German, "select_host", "\nHost auswählen (1-%d): ")
	message.SetString(language.French, "select_host", "\nSélectionner l'hôte (1-%d): ")
	
	message.SetString(language.English, "invalid_selection", "invalid selection")
	message.SetString(language.Spanish, "invalid_selection", "selección inválida")
	message.SetString(language.Chinese, "invalid_selection", "无效选择")
	message.SetString(language.Hindi, "invalid_selection", "अमान्य चयन")
	message.SetString(language.Arabic, "invalid_selection", "اختيار غير صحيح")
	message.SetString(language.Bengali, "invalid_selection", "অবৈধ নির্বাচন")
	message.SetString(language.Portuguese, "invalid_selection", "seleção inválida")
	message.SetString(language.Russian, "invalid_selection", "неверный выбор")
	message.SetString(language.Japanese, "invalid_selection", "無効な選択")
	message.SetString(language.German, "invalid_selection", "ungültige Auswahl")
	message.SetString(language.French, "invalid_selection", "sélection invalide")
	
	message.SetString(language.English, "selection_out_of_range", "selection out of range")
	message.SetString(language.Spanish, "selection_out_of_range", "selección fuera de rango")
	message.SetString(language.Chinese, "selection_out_of_range", "选择超出范围")
	message.SetString(language.Hindi, "selection_out_of_range", "चयन सीमा से बाहर")
	message.SetString(language.Arabic, "selection_out_of_range", "الاختيار خارج النطاق")
	message.SetString(language.Bengali, "selection_out_of_range", "নির্বাচন পরিসরের বাইরে")
	message.SetString(language.Portuguese, "selection_out_of_range", "seleção fora do intervalo")
	message.SetString(language.Russian, "selection_out_of_range", "выбор вне диапазона")
	message.SetString(language.Japanese, "selection_out_of_range", "選択が範囲外")
	message.SetString(language.German, "selection_out_of_range", "Auswahl außerhalb des Bereichs")
	message.SetString(language.French, "selection_out_of_range", "sélection hors plage")
	
	message.SetString(language.English, "connecting_to", "Connecting to %s...")
	message.SetString(language.Spanish, "connecting_to", "Conectando a %s...")
	message.SetString(language.Chinese, "connecting_to", "正在连接到 %s...")
	message.SetString(language.Hindi, "connecting_to", "%s से कनेक्ट हो रहे हैं...")
	message.SetString(language.Arabic, "connecting_to", "الاتصال بـ %s...")
	message.SetString(language.Bengali, "connecting_to", "%s এ সংযোগ করা হচ্ছে...")
	message.SetString(language.Portuguese, "connecting_to", "Conectando a %s...")
	message.SetString(language.Russian, "connecting_to", "Подключение к %s...")
	message.SetString(language.Japanese, "connecting_to", "%s に接続中...")
	message.SetString(language.German, "connecting_to", "Verbindung zu %s...")
	message.SetString(language.French, "connecting_to", "Connexion à %s...")
	
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
	message.SetString(language.Chinese, "copy_failed", "复制到 %s 失败: %v")
	message.SetString(language.Hindi, "copy_failed", "%s में कॉपी करना विफल: %v")
	message.SetString(language.Arabic, "copy_failed", "فشل في النسخ إلى %s: %v")
	message.SetString(language.Bengali, "copy_failed", "%s এ কপি করতে ব্যর্থ: %v")
	message.SetString(language.Portuguese, "copy_failed", "Falha ao copiar para %s: %v")
	message.SetString(language.Russian, "copy_failed", "Не удалось скопировать в %s: %v")
	message.SetString(language.Japanese, "copy_failed", "%s への複写に失敗: %v")
	message.SetString(language.German, "copy_failed", "Fehler beim Kopieren nach %s: %v")
	message.SetString(language.French, "copy_failed", "Échec de la copie vers %s: %v")
	
	message.SetString(language.English, "copy_success", "Successfully copied to %s")
	message.SetString(language.Spanish, "copy_success", "Copiado exitosamente a %s")
	message.SetString(language.Chinese, "copy_success", "成功复制到 %s")
	message.SetString(language.Hindi, "copy_success", "%s में सफलतापूर्वक कॉपी की गई")
	message.SetString(language.Arabic, "copy_success", "تم النسخ بنجاح إلى %s")
	message.SetString(language.Bengali, "copy_success", "%s এ সফলভাবে কপি করা হয়েছে")
	message.SetString(language.Portuguese, "copy_success", "Copiado com sucesso para %s")
	message.SetString(language.Russian, "copy_success", "Успешно скопировано в %s")
	message.SetString(language.Japanese, "copy_success", "%s への複写が成功しました")
	message.SetString(language.German, "copy_success", "Erfolgreich nach %s kopiert")
	message.SetString(language.French, "copy_success", "Copié avec succès vers %s")
	
	// SCP error messages
	message.SetString(language.English, "invalid_scp_remote", "invalid remote SCP argument format: %q. Must be [user@]host:path")
	message.SetString(language.Spanish, "invalid_scp_remote", "formato de argumento SCP remoto inválido: %q. Debe ser [usuario@]servidor:ruta")
	
	message.SetString(language.English, "invalid_user_host", "invalid user@host format in SCP argument: %q")
	message.SetString(language.Spanish, "invalid_user_host", "formato usuario@servidor inválido en argumento SCP: %q")
	
	message.SetString(language.English, "empty_host_scp", "host cannot be empty in SCP argument: %q")
	message.SetString(language.Spanish, "empty_host_scp", "el servidor no puede estar vacío en argumento SCP: %q")
	
	// Flag descriptions
	message.SetString(language.English, "flag_lang_desc", "Language for CLI output (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.Spanish, "flag_lang_desc", "Idioma para salida CLI (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.Chinese, "flag_lang_desc", "CLI输出语言 (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.Hindi, "flag_lang_desc", "CLI आउटपुट के लिए भाषा (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.Arabic, "flag_lang_desc", "لغة إخراج CLI (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.Bengali, "flag_lang_desc", "CLI আউটপুটের ভাষা (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.Portuguese, "flag_lang_desc", "Idioma para saída CLI (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.Russian, "flag_lang_desc", "Язык для вывода CLI (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.Japanese, "flag_lang_desc", "CLI出力の言語 (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.German, "flag_lang_desc", "Sprache für CLI-Ausgabe (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	message.SetString(language.French, "flag_lang_desc", "Langue pour la sortie CLI (en, es, zh, hi, ar, bn, pt, ru, ja, de, fr)")
	
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
	message.SetString(language.Chinese, "cli_description", "具有 Tailscale 连接的企业级安全 SSH/SCP 客户端")
	message.SetString(language.Hindi, "cli_description", "एंटरप्राइज़ वातावरण के लिए Tailscale कनेक्टिविटी के साथ सुरक्षित SSH/SCP क्लाइंट")
	message.SetString(language.Arabic, "cli_description", "عميل SSH/SCP آمن مع اتصال Tailscale للبيئات المؤسسية")
	message.SetString(language.Bengali, "cli_description", "এন্টারপ্রাইজ পরিবেশের জন্য Tailscale সংযোগ সহ নিরাপদ SSH/SCP ক্লায়েন্ট")
	message.SetString(language.Portuguese, "cli_description", "Cliente SSH/SCP seguro com conectividade Tailscale para ambientes empresariais")
	message.SetString(language.Russian, "cli_description", "Безопасный SSH/SCP клиент с подключением Tailscale для корпоративных сред")
	message.SetString(language.Japanese, "cli_description", "企業環境向けTailscale接続対応セキュアSSH/SCPクライアント")
	message.SetString(language.German, "cli_description", "Sicherer SSH/SCP-Client mit Tailscale-Konnektivität für Unternehmensumgebungen")
	message.SetString(language.French, "cli_description", "Client SSH/SCP sécurisé avec connectivité Tailscale pour environnements d'entreprise")
	
	message.SetString(language.English, "cmd_connect_desc", "Connect to a remote host via SSH (default command)")
	message.SetString(language.Spanish, "cmd_connect_desc", "Conectar a un servidor remoto via SSH (comando por defecto)")
	message.SetString(language.Chinese, "cmd_connect_desc", "通过 SSH 连接到远程主机（默认命令）")
	message.SetString(language.Hindi, "cmd_connect_desc", "SSH के माध्यम से रिमोट होस्ट से कनेक्ट करें (डिफ़ॉल्ट कमांड)")
	message.SetString(language.Arabic, "cmd_connect_desc", "الاتصال بمضيف بعيد عبر SSH (الأمر الافتراضي)")
	message.SetString(language.Bengali, "cmd_connect_desc", "SSH এর মাধ্যমে রিমোট হোস্টে সংযোগ করুন (ডিফল্ট কমান্ড)")
	message.SetString(language.Portuguese, "cmd_connect_desc", "Conectar a um host remoto via SSH (comando padrão)")
	message.SetString(language.Russian, "cmd_connect_desc", "Подключиться к удаленному хосту через SSH (команда по умолчанию)")
	message.SetString(language.Japanese, "cmd_connect_desc", "SSH経由でリモートホストに接続（デフォルトコマンド）")
	message.SetString(language.German, "cmd_connect_desc", "Verbindung zu einem Remote-Host über SSH (Standardbefehl)")
	message.SetString(language.French, "cmd_connect_desc", "Se connecter à un hôte distant via SSH (commande par défaut)")
	
	message.SetString(language.English, "cmd_scp_desc", "Transfer files securely using SCP")
	message.SetString(language.Spanish, "cmd_scp_desc", "Transferir archivos de forma segura usando SCP")
	message.SetString(language.Chinese, "cmd_scp_desc", "使用 SCP 安全传输文件")
	message.SetString(language.Hindi, "cmd_scp_desc", "SCP का उपयोग करके फाइलों को सुरक्षित रूप से स्थानांतरित करें")
	message.SetString(language.Arabic, "cmd_scp_desc", "نقل الملفات بأمان باستخدام SCP")
	message.SetString(language.Bengali, "cmd_scp_desc", "SCP ব্যবহার করে নিরাপদে ফাইল স্থানান্তর")
	message.SetString(language.Portuguese, "cmd_scp_desc", "Transferir arquivos com segurança usando SCP")
	message.SetString(language.Russian, "cmd_scp_desc", "Безопасная передача файлов с помощью SCP")
	message.SetString(language.Japanese, "cmd_scp_desc", "SCPを使用したセキュアファイル転送")
	message.SetString(language.German, "cmd_scp_desc", "Dateien sicher mit SCP übertragen")
	message.SetString(language.French, "cmd_scp_desc", "Transférer des fichiers en toute sécurité avec SCP")
	
	message.SetString(language.English, "cmd_list_desc", "List available Tailscale hosts")
	message.SetString(language.Spanish, "cmd_list_desc", "Listar servidores Tailscale disponibles")
	message.SetString(language.Chinese, "cmd_list_desc", "列出可用的 Tailscale 主机")
	message.SetString(language.Hindi, "cmd_list_desc", "उपलब्ध Tailscale होस्ट की सूची बनाएं")
	message.SetString(language.Arabic, "cmd_list_desc", "سرد مضيفي Tailscale المتاحين")
	message.SetString(language.Bengali, "cmd_list_desc", "উপলব্ধ Tailscale হোস্টের তালিকা")
	message.SetString(language.Portuguese, "cmd_list_desc", "Listar hosts Tailscale disponíveis")
	message.SetString(language.Russian, "cmd_list_desc", "Показать доступные Tailscale хосты")
	message.SetString(language.Japanese, "cmd_list_desc", "利用可能なTailscaleホストを一覧表示")
	message.SetString(language.German, "cmd_list_desc", "Verfügbare Tailscale-Hosts auflisten")
	message.SetString(language.French, "cmd_list_desc", "Lister les hôtes Tailscale disponibles")
	
	message.SetString(language.English, "cmd_exec_desc", "Execute commands on multiple hosts")
	message.SetString(language.Spanish, "cmd_exec_desc", "Ejecutar comandos en múltiples servidores")
	message.SetString(language.Chinese, "cmd_exec_desc", "在多个主机上执行命令")
	message.SetString(language.Hindi, "cmd_exec_desc", "कई होस्ट पर कमांड का कार्यान्वयन")
	message.SetString(language.Arabic, "cmd_exec_desc", "تنفيذ الأوامر على عدة مضيفين")
	message.SetString(language.Bengali, "cmd_exec_desc", "একাধিক হোস্টে কমান্ড নিষ্পাদন")
	message.SetString(language.Portuguese, "cmd_exec_desc", "Executar comandos em vários hosts")
	message.SetString(language.Russian, "cmd_exec_desc", "Выполнить команды на нескольких хостах")
	message.SetString(language.Japanese, "cmd_exec_desc", "複数のホストでコマンドを実行")
	message.SetString(language.German, "cmd_exec_desc", "Befehle auf mehreren Hosts ausführen")
	message.SetString(language.French, "cmd_exec_desc", "Exécuter des commandes sur plusieurs hôtes")
	
	message.SetString(language.English, "cmd_multi_desc", "Multi-host operations with tmux session management")
	message.SetString(language.Spanish, "cmd_multi_desc", "Operaciones multi-servidor con gestión de sesiones tmux")
	message.SetString(language.Chinese, "cmd_multi_desc", "使用 tmux 会话管理的多主机操作")
	message.SetString(language.Hindi, "cmd_multi_desc", "tmux सत्र प्रबंधन के साथ मल्टी-होस्ट ऑपरेशन")
	message.SetString(language.Arabic, "cmd_multi_desc", "عمليات متعددة المضيفين مع إدارة جلسات tmux")
	message.SetString(language.Bengali, "cmd_multi_desc", "tmux সেশন পরিচালনা সহ মাল্টি-হোস্ট অপারেশন")
	message.SetString(language.Portuguese, "cmd_multi_desc", "Operações multi-host com gerenciamento de sessão tmux")
	message.SetString(language.Russian, "cmd_multi_desc", "Мульти-хост операции с управлением сессий tmux")
	message.SetString(language.Japanese, "cmd_multi_desc", "tmuxセッション管理を伴ったマルチホスト操作")
	message.SetString(language.German, "cmd_multi_desc", "Multi-Host-Operationen mit tmux-Sitzungsverwaltung")
	message.SetString(language.French, "cmd_multi_desc", "Opérations multi-hôtes avec gestion de session tmux")
	
	message.SetString(language.English, "cmd_config_desc", "Manage application configuration")
	message.SetString(language.Spanish, "cmd_config_desc", "Gestionar configuración de la aplicación")
	message.SetString(language.Chinese, "cmd_config_desc", "管理应用程序配置")
	message.SetString(language.Hindi, "cmd_config_desc", "एप्लिकेशन कॉन्फ़िगरेशन का प्रबंधन")
	message.SetString(language.Arabic, "cmd_config_desc", "إدارة تكوين التطبيق")
	message.SetString(language.Bengali, "cmd_config_desc", "অ্যাপ্লিকেশন কনফিগারেশন পরিচালনা")
	message.SetString(language.Portuguese, "cmd_config_desc", "Gerenciar configuração da aplicação")
	message.SetString(language.Russian, "cmd_config_desc", "Управление конфигурацией приложения")
	message.SetString(language.Japanese, "cmd_config_desc", "アプリケーション設定の管理")
	message.SetString(language.German, "cmd_config_desc", "Anwendungskonfiguration verwalten")
	message.SetString(language.French, "cmd_config_desc", "Gérer la configuration de l'application")
	
	message.SetString(language.English, "cmd_pqc_desc", "Post-quantum cryptography operations and reporting")
	message.SetString(language.Spanish, "cmd_pqc_desc", "Operaciones y reportes de criptografía post-cuántica")
	message.SetString(language.Chinese, "cmd_pqc_desc", "后量子密码学操作和报告")
	message.SetString(language.Hindi, "cmd_pqc_desc", "पोस्ट-क्वांटम क्रिप्टोग्राफी ऑपरेशन और रिपोर्टिंग")
	message.SetString(language.Arabic, "cmd_pqc_desc", "عمليات وتقارير التشفير ما بعد الكمي")
	message.SetString(language.Bengali, "cmd_pqc_desc", "পোস্ট-কোয়ান্টাম ক্রিপ্টোগ্রাফি অপারেশন এবং রিপোর্টিং")
	message.SetString(language.Portuguese, "cmd_pqc_desc", "Operações e relatórios de criptografia pós-quântica")
	message.SetString(language.Russian, "cmd_pqc_desc", "Операции и отчеты постквантовой криптографии")
	message.SetString(language.Japanese, "cmd_pqc_desc", "ポスト量子暗号の操作とレポート")
	message.SetString(language.German, "cmd_pqc_desc", "Post-Quanten-Kryptographie-Operationen und Berichte")
	message.SetString(language.French, "cmd_pqc_desc", "Opérations et rapports de cryptographie post-quantique")
	
	message.SetString(language.English, "cmd_version_desc", "Show version information")
	message.SetString(language.Spanish, "cmd_version_desc", "Mostrar información de versión")
	message.SetString(language.Chinese, "cmd_version_desc", "显示版本信息")
	message.SetString(language.Hindi, "cmd_version_desc", "वर्जन जानकारी दिखाएं")
	message.SetString(language.Arabic, "cmd_version_desc", "عرض معلومات الإصدار")
	message.SetString(language.Bengali, "cmd_version_desc", "ভার্সনের তথ্য দেখান")
	message.SetString(language.Portuguese, "cmd_version_desc", "Mostrar informações de versão")
	message.SetString(language.Russian, "cmd_version_desc", "Показать информацию о версии")
	message.SetString(language.Japanese, "cmd_version_desc", "バージョン情報を表示")
	message.SetString(language.German, "cmd_version_desc", "Versionsinformationen anzeigen")
	message.SetString(language.French, "cmd_version_desc", "Afficher les informations de version")

	// CLI Short and Long descriptions for Cobra/Fang
	message.SetString(language.English, "root_short", "SSH client with Tailscale integration")
	message.SetString(language.Spanish, "root_short", "Cliente SSH con integración Tailscale")
	message.SetString(language.Chinese, "root_short", "具有 Tailscale 集成的 SSH 客户端")
	message.SetString(language.Hindi, "root_short", "Tailscale इंटीग्रेशन के साथ SSH क्लाइंट")
	message.SetString(language.Arabic, "root_short", "عميل SSH مع تكامل Tailscale")
	message.SetString(language.Bengali, "root_short", "Tailscale ইন্টিগ্রেশন সহ SSH ক্লায়েন্ট")
	message.SetString(language.Portuguese, "root_short", "Cliente SSH com integração Tailscale")
	message.SetString(language.Russian, "root_short", "SSH-клиент с интеграцией Tailscale")
	message.SetString(language.Japanese, "root_short", "Tailscale統合SSHクライアント")
	message.SetString(language.German, "root_short", "SSH-Client mit Tailscale-Integration")
	message.SetString(language.French, "root_short", "Client SSH avec intégration Tailscale")

	message.SetString(language.English, "root_long", "A secure SSH client that works seamlessly with Tailscale networks")
	message.SetString(language.Spanish, "root_long", "Un cliente SSH seguro que funciona perfectamente con redes Tailscale")
	message.SetString(language.Chinese, "root_long", "与 Tailscale 网络无缝协作的安全 SSH 客户端")
	message.SetString(language.Hindi, "root_long", "एक सुरक्षित SSH क्लाइंट जो Tailscale नेटवर्क के साथ बेहतरीन काम करता है")
	message.SetString(language.Arabic, "root_long", "عميل SSH آمن يعمل بسلاسة مع شبكات Tailscale")
	message.SetString(language.Bengali, "root_long", "একটি নিরাপদ SSH ক্লায়েন্ট যা Tailscale নেটওয়ার্কের সাথে নির্বিঘ্নে কাজ করে")
	message.SetString(language.Portuguese, "root_long", "Um cliente SSH seguro que funciona perfeitamente com redes Tailscale")
	message.SetString(language.Russian, "root_long", "Безопасный SSH-клиент, который беспрепятственно работает с сетями Tailscale")
	message.SetString(language.Japanese, "root_long", "Tailscaleネットワークとシームレスに連携するセキュアSSHクライアント")
	message.SetString(language.German, "root_long", "Ein sicherer SSH-Client, der nahtlos mit Tailscale-Netzwerken funktioniert")
	message.SetString(language.French, "root_long", "Un client SSH sécurisé qui fonctionne parfaitement avec les réseaux Tailscale")

	message.SetString(language.English, "root_examples", `  # Connect to a host
  ts-ssh user@hostname
  
  # Execute a command
  ts-ssh hostname "ls -la"
  
  # Copy files with SCP
  ts-ssh scp local.txt user@host:/remote/path/
  
  # List available hosts
  ts-ssh list
  
  # Interactive host selection
  ts-ssh list --interactive`)
	message.SetString(language.Spanish, "root_examples", `  # Conectar a un servidor
  ts-ssh usuario@servidor
  
  # Ejecutar un comando
  ts-ssh servidor "ls -la"
  
  # Copiar archivos con SCP
  ts-ssh scp archivo.txt usuario@servidor:/ruta/remota/
  
  # Listar servidores disponibles
  ts-ssh list
  
  # Selección interactiva de servidor
  ts-ssh list --interactive`)
	message.SetString(language.Chinese, "root_examples", `  # 连接到主机
  ts-ssh 用户@主机名
  
  # 执行命令
  ts-ssh 主机名 "ls -la"
  
  # 使用 SCP 复制文件
  ts-ssh scp 本地文件.txt 用户@主机:/远程/路径/
  
  # 列出可用主机
  ts-ssh list
  
  # 交互式主机选择
  ts-ssh list --interactive`)
	message.SetString(language.German, "root_examples", `  # Verbindung zu einem Host
  ts-ssh benutzer@hostname
  
  # Befehl ausführen
  ts-ssh hostname "ls -la"
  
  # Dateien mit SCP kopieren
  ts-ssh scp datei.txt benutzer@host:/remote/pfad/
  
  # Verfügbare Hosts auflisten
  ts-ssh list
  
  # Interaktive Host-Auswahl
  ts-ssh list --interactive`)
	message.SetString(language.French, "root_examples", `  # Se connecter à un hôte
  ts-ssh utilisateur@hostname
  
  # Exécuter une commande
  ts-ssh hostname "ls -la"
  
  # Copier des fichiers avec SCP
  ts-ssh scp fichier.txt utilisateur@hôte:/chemin/distant/
  
  # Lister les hôtes disponibles
  ts-ssh list
  
  # Sélection interactive d'hôte
  ts-ssh list --interactive`)

	// Connect command
	message.SetString(language.English, "connect_short", "Connect to a host via SSH")
	message.SetString(language.Spanish, "connect_short", "Conectar a un servidor via SSH")
	message.SetString(language.Chinese, "connect_short", "通过 SSH 连接到主机")
	message.SetString(language.Hindi, "connect_short", "SSH के माध्यम से होस्ट से कनेक्ट करें")
	message.SetString(language.Arabic, "connect_short", "الاتصال بمضيف عبر SSH")
	message.SetString(language.Bengali, "connect_short", "SSH এর মাধ্যমে হোস্টে সংযুক্ত হন")
	message.SetString(language.Portuguese, "connect_short", "Conectar a um host via SSH")
	message.SetString(language.Russian, "connect_short", "Подключиться к хосту через SSH")
	message.SetString(language.Japanese, "connect_short", "SSH経由でホストに接続")
	message.SetString(language.German, "connect_short", "Verbindung zu einem Host über SSH")
	message.SetString(language.French, "connect_short", "Se connecter à un hôte via SSH")

	message.SetString(language.English, "connect_long", "Establish an SSH connection to a remote host through Tailscale")
	message.SetString(language.Spanish, "connect_long", "Establecer una conexión SSH a un servidor remoto a través de Tailscale")
	message.SetString(language.Chinese, "connect_long", "通过 Tailscale 建立到远程主机的 SSH 连接")
	message.SetString(language.Hindi, "connect_long", "Tailscale के माध्यम से रिमोट होस्ट से SSH कनेक्शन स्थापित करें")
	message.SetString(language.Arabic, "connect_long", "إنشاء اتصال SSH إلى مضيف بعيد عبر Tailscale")
	message.SetString(language.Bengali, "connect_long", "Tailscale এর মাধ্যমে একটি রিমোট হোস্টে SSH কনেকশন স্থাপন করুন")
	message.SetString(language.Portuguese, "connect_long", "Estabelecer uma conexão SSH para um host remoto através do Tailscale")
	message.SetString(language.Russian, "connect_long", "Установить SSH-соединение с удаленным хостом через Tailscale")
	message.SetString(language.Japanese, "connect_long", "Tailscale経由でリモートホストへのSSH接続を確立")
	message.SetString(language.German, "connect_long", "SSH-Verbindung zu einem entfernten Host über Tailscale herstellen")
	message.SetString(language.French, "connect_long", "Établir une connexion SSH vers un hôte distant via Tailscale")

	message.SetString(language.English, "connect_examples", `  # Simple connection
  ts-ssh connect user@hostname
  
  # Execute remote command
  ts-ssh connect hostname "uptime"
  
  # Port forwarding
  ts-ssh connect -W dest:port hostname`)
	message.SetString(language.Chinese, "connect_examples", `  # 简单连接
  ts-ssh connect 用户@主机名
  
  # 执行远程命令
  ts-ssh connect 主机名 "uptime"
  
  # 端口转发
  ts-ssh connect -W 目标:端口 主机名`)
	message.SetString(language.German, "connect_examples", `  # Einfache Verbindung
  ts-ssh connect benutzer@hostname
  
  # Remote-Befehl ausführen
  ts-ssh connect hostname "uptime"
  
  # Port-Weiterleitung
  ts-ssh connect -W ziel:port hostname`)
	message.SetString(language.French, "connect_examples", `  # Connexion simple
  ts-ssh connect utilisateur@hostname
  
  # Exécuter une commande distante
  ts-ssh connect hostname "uptime"
  
  # Redirection de port
  ts-ssh connect -W destination:port hostname`)

	// SCP command
	message.SetString(language.English, "scp_short", "Copy files via SCP")
	message.SetString(language.Spanish, "scp_short", "Copiar archivos via SCP")
	message.SetString(language.Chinese, "scp_short", "通过 SCP 复制文件")
	message.SetString(language.Hindi, "scp_short", "SCP के माध्यम से फाइलें कॉपी करें")
	message.SetString(language.Arabic, "scp_short", "نسخ الملفات عبر SCP")
	message.SetString(language.Bengali, "scp_short", "SCP এর মাধ্যমে ফাইল কপি করুন")
	message.SetString(language.Portuguese, "scp_short", "Copiar arquivos via SCP")
	message.SetString(language.Russian, "scp_short", "Копировать файлы через SCP")
	message.SetString(language.Japanese, "scp_short", "SCP経由でファイルをコピー")
	message.SetString(language.German, "scp_short", "Dateien über SCP kopieren")
	message.SetString(language.French, "scp_short", "Copier des fichiers via SCP")

	message.SetString(language.English, "scp_long", "Securely copy files between local and remote hosts using SCP protocol")
	message.SetString(language.Spanish, "scp_long", "Copiar archivos de forma segura entre hosts locales y remotos usando protocolo SCP")
	message.SetString(language.Chinese, "scp_long", "使用 SCP 协议在本地和远程主机之间安全复制文件")
	message.SetString(language.Hindi, "scp_long", "SCP प्रोटोकॉल का उपयोग करके स्थानीय और दूरस्थ होस्ट के बीच फाइलों को सुरक्षित रूप से कॉपी करें")
	message.SetString(language.Arabic, "scp_long", "نسخ آمن للملفات بين المضيفين المحليين والبعيدين باستخدام بروتوكول SCP")
	message.SetString(language.Bengali, "scp_long", "SCP প্রোটোকল ব্যবহার করে স্থানীয় এবং দূরবর্তী হোস্টের মধ্যে নিরাপদে ফাইল কপি করুন")
	message.SetString(language.Portuguese, "scp_long", "Copiar arquivos com segurança entre hosts locais e remotos usando protocolo SCP")
	message.SetString(language.Russian, "scp_long", "Безопасное копирование файлов между локальными и удаленными хостами с использованием протокола SCP")
	message.SetString(language.Japanese, "scp_long", "SCPプロトコルを使用してローカルとリモートホスト間でファイルを安全にコピー")
	message.SetString(language.German, "scp_long", "Sichere Übertragung von Dateien zwischen lokalen und entfernten Hosts mit SCP-Protokoll")
	message.SetString(language.French, "scp_long", "Copier en toute sécurité des fichiers entre hôtes locaux et distants en utilisant le protocole SCP")

	message.SetString(language.English, "scp_examples", `  # Copy local to remote
  ts-ssh scp local.txt user@host:/path/
  
  # Copy remote to local
  ts-ssh scp user@host:/path/file.txt ./
  
  # Recursive copy
  ts-ssh scp -r ./directory/ user@host:/path/`)
	message.SetString(language.Chinese, "scp_examples", `  # 本地复制到远程
  ts-ssh scp local.txt 用户@主机:/路径/
  
  # 远程复制到本地
  ts-ssh scp 用户@主机:/路径/文件.txt ./
  
  # 递归复制
  ts-ssh scp -r ./目录/ 用户@主机:/路径/`)
	message.SetString(language.German, "scp_examples", `  # Lokal zu Remote kopieren
  ts-ssh scp local.txt benutzer@host:/pfad/
  
  # Remote zu Lokal kopieren
  ts-ssh scp benutzer@host:/pfad/datei.txt ./
  
  # Rekursiv kopieren
  ts-ssh scp -r ./verzeichnis/ benutzer@host:/pfad/`)
	message.SetString(language.French, "scp_examples", `  # Copier local vers distant
  ts-ssh scp local.txt utilisateur@hôte:/chemin/
  
  # Copier distant vers local
  ts-ssh scp utilisateur@hôte:/chemin/fichier.txt ./
  
  # Copie récursive
  ts-ssh scp -r ./répertoire/ utilisateur@hôte:/chemin/`)

	// List command
	message.SetString(language.English, "list_short", "List available hosts")
	message.SetString(language.Spanish, "list_short", "Listar hosts disponibles")
	message.SetString(language.Chinese, "list_short", "列出可用主机")
	message.SetString(language.Hindi, "list_short", "उपलब्ध होस्ट सूची")
	message.SetString(language.Arabic, "list_short", "عرض المضيفين المتاحين")
	message.SetString(language.Bengali, "list_short", "উপলব্ধ হোস্ট তালিকা")
	message.SetString(language.Portuguese, "list_short", "Listar hosts disponíveis")
	message.SetString(language.Russian, "list_short", "Список доступных хостов")
	message.SetString(language.Japanese, "list_short", "利用可能なホストをリスト")
	message.SetString(language.German, "list_short", "Verfügbare Hosts auflisten")
	message.SetString(language.French, "list_short", "Lister les hôtes disponibles")

	message.SetString(language.English, "list_long", "Display all available hosts on the Tailscale network")
	message.SetString(language.Spanish, "list_long", "Mostrar todos los hosts disponibles en la red Tailscale")
	message.SetString(language.Chinese, "list_long", "显示 Tailscale 网络上所有可用的主机")
	message.SetString(language.Hindi, "list_long", "Tailscale नेटवर्क पर सभी उपलब्ध होस्ट प्रदर्शित करें")
	message.SetString(language.Arabic, "list_long", "عرض جميع المضيفين المتاحين على شبكة Tailscale")
	message.SetString(language.Bengali, "list_long", "Tailscale নেটওয়ার্কে সমস্ত উপলব্ধ হোস্ট প্রদর্শন করুন")
	message.SetString(language.Portuguese, "list_long", "Exibir todos os hosts disponíveis na rede Tailscale")
	message.SetString(language.Russian, "list_long", "Отобразить все доступные хосты в сети Tailscale")
	message.SetString(language.Japanese, "list_long", "Tailscaleネットワーク上のすべての利用可能なホストを表示")
	message.SetString(language.German, "list_long", "Alle verfügbaren Hosts im Tailscale-Netzwerk anzeigen")
	message.SetString(language.French, "list_long", "Afficher tous les hôtes disponibles sur le réseau Tailscale")

	message.SetString(language.English, "list_examples", `  # List all hosts
  ts-ssh list
  
  # Interactive host selection
  ts-ssh list --interactive`)
	message.SetString(language.Spanish, "list_examples", `  # Listar todos los hosts
  ts-ssh list
  
  # Selección interactiva de hosts
  ts-ssh list --interactive`)
	message.SetString(language.Chinese, "list_examples", `  # 列出所有主机
  ts-ssh list
  
  # 交互式主机选择
  ts-ssh list --interactive`)
	message.SetString(language.Hindi, "list_examples", `  # सभी होस्ट सूची
  ts-ssh list
  
  # इंटरैक्टिव होस्ट चयन
  ts-ssh list --interactive`)
	message.SetString(language.Arabic, "list_examples", `  # عرض جميع المضيفين
  ts-ssh list
  
  # اختيار تفاعلي للمضيف
  ts-ssh list --interactive`)
	message.SetString(language.Bengali, "list_examples", `  # সমস্ত হোস্ট তালিকা
  ts-ssh list
  
  # ইন্টারঅ্যাক্টিভ হোস্ট নির্বাচন
  ts-ssh list --interactive`)
	message.SetString(language.Portuguese, "list_examples", `  # Listar todos os hosts
  ts-ssh list
  
  # Seleção interativa de hosts
  ts-ssh list --interactive`)
	message.SetString(language.Russian, "list_examples", `  # Список всех хостов
  ts-ssh list
  
  # Интерактивный выбор хоста
  ts-ssh list --interactive`)
	message.SetString(language.Japanese, "list_examples", `  # すべてのホストをリスト
  ts-ssh list
  
  # インタラクティブなホスト選択
  ts-ssh list --interactive`)
	message.SetString(language.German, "list_examples", `  # Alle Hosts auflisten
  ts-ssh list
  
  # Interaktive Host-Auswahl
  ts-ssh list --interactive`)
	message.SetString(language.French, "list_examples", `  # Lister tous les hôtes
  ts-ssh list
  
  # Sélection interactive d'hôte
  ts-ssh list --interactive`)

	// Exec command
	message.SetString(language.English, "exec_short", "Execute commands on multiple hosts")
	message.SetString(language.Spanish, "exec_short", "Ejecutar comandos en múltiples hosts")
	message.SetString(language.Chinese, "exec_short", "在多个主机上执行命令")
	message.SetString(language.Hindi, "exec_short", "कई होस्ट पर कमांड चलाएं")
	message.SetString(language.Arabic, "exec_short", "تنفيذ الأوامر على عدة مضيفين")
	message.SetString(language.Bengali, "exec_short", "একাধিক হোস্টে কমান্ড চালান")
	message.SetString(language.Portuguese, "exec_short", "Executar comandos em múltiplos hosts")
	message.SetString(language.Russian, "exec_short", "Выполнить команды на нескольких хостах")
	message.SetString(language.Japanese, "exec_short", "複数のホストでコマンド実行")
	message.SetString(language.German, "exec_short", "Befehle auf mehreren Hosts ausführen")
	message.SetString(language.French, "exec_short", "Exécuter des commandes sur plusieurs hôtes")

	message.SetString(language.English, "exec_long", "Run the same command across multiple hosts simultaneously")
	message.SetString(language.Spanish, "exec_long", "Ejecutar el mismo comando en múltiples hosts simultáneamente")
	message.SetString(language.Chinese, "exec_long", "同时在多个主机上运行相同的命令")
	message.SetString(language.Hindi, "exec_long", "एक साथ कई होस्ट पर एक ही कमांड चलाएं")
	message.SetString(language.Arabic, "exec_long", "تشغيل نفس الأمر عبر عدة مضيفين في نفس الوقت")
	message.SetString(language.Bengali, "exec_long", "একই সাথে একাধিক হোস্টে একই কমান্ড চালান")
	message.SetString(language.Portuguese, "exec_long", "Executar o mesmo comando em múltiplos hosts simultaneamente")
	message.SetString(language.Russian, "exec_long", "Запустить одну и ту же команду на нескольких хостах одновременно")
	message.SetString(language.Japanese, "exec_long", "複数のホストで同じコマンドを同時に実行")
	message.SetString(language.German, "exec_long", "Denselben Befehl gleichzeitig auf mehreren Hosts ausführen")
	message.SetString(language.French, "exec_long", "Exécuter la même commande sur plusieurs hôtes simultanément")

	message.SetString(language.English, "exec_examples", `  # Execute on specific hosts
  ts-ssh exec host1 host2 -c "uptime"
  
  # Execute in parallel
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.Spanish, "exec_examples", `  # Ejecutar en hosts específicos
  ts-ssh exec host1 host2 -c "uptime"
  
  # Ejecutar en paralelo
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.Chinese, "exec_examples", `  # 在特定主机上执行
  ts-ssh exec host1 host2 -c "uptime"
  
  # 并行执行
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.Hindi, "exec_examples", `  # विशिष्ट होस्ट पर चलाएं
  ts-ssh exec host1 host2 -c "uptime"
  
  # समानांतर में चलाएं
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.Arabic, "exec_examples", `  # تنفيذ على مضيفين محددين
  ts-ssh exec host1 host2 -c "uptime"
  
  # تنفيذ متوازي
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.Bengali, "exec_examples", `  # নির্দিষ্ট হোস্টে চালান
  ts-ssh exec host1 host2 -c "uptime"
  
  # সমান্তরালে চালান
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.Portuguese, "exec_examples", `  # Executar em hosts específicos
  ts-ssh exec host1 host2 -c "uptime"
  
  # Executar em paralelo
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.Russian, "exec_examples", `  # Выполнить на конкретных хостах
  ts-ssh exec host1 host2 -c "uptime"
  
  # Выполнить параллельно
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.Japanese, "exec_examples", `  # 特定のホストで実行
  ts-ssh exec host1 host2 -c "uptime"
  
  # 並列実行
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.German, "exec_examples", `  # Auf bestimmten Hosts ausführen
  ts-ssh exec host1 host2 -c "uptime"
  
  # Parallel ausführen
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)
	message.SetString(language.French, "exec_examples", `  # Exécuter sur des hôtes spécifiques
  ts-ssh exec host1 host2 -c "uptime"
  
  # Exécuter en parallèle
  ts-ssh exec host1 host2 host3 -c "df -h" --parallel`)

	// Multi command
	message.SetString(language.English, "multi_short", "Handle multi-host operations")
	message.SetString(language.Spanish, "multi_short", "Manejar operaciones multi-host")
	message.SetString(language.Chinese, "multi_short", "处理多主机操作")
	message.SetString(language.Hindi, "multi_short", "मल्टी-होस्ट ऑपरेशन्स हैंडल करें")
	message.SetString(language.Arabic, "multi_short", "التعامل مع عمليات المضيفين المتعددين")
	message.SetString(language.Bengali, "multi_short", "মাল্টি-হোস্ট অপারেশন পরিচালনা")
	message.SetString(language.Portuguese, "multi_short", "Gerenciar operações multi-host")
	message.SetString(language.Russian, "multi_short", "Обработка операций с несколькими хостами")
	message.SetString(language.Japanese, "multi_short", "マルチホスト操作の処理")
	message.SetString(language.German, "multi_short", "Multi-Host-Operationen verwalten")
	message.SetString(language.French, "multi_short", "Gérer les opérations multi-hôtes")

	message.SetString(language.English, "multi_long", "Manage connections to multiple hosts with advanced session handling")
	message.SetString(language.Spanish, "multi_long", "Gestionar conexiones a múltiples hosts con manejo avanzado de sesiones")
	message.SetString(language.Chinese, "multi_long", "使用高级会话处理管理到多个主机的连接")
	message.SetString(language.Hindi, "multi_long", "उन्नत सेशन हैंडलिंग के साथ कई होस्ट के कनेक्शन प्रबंधित करें")
	message.SetString(language.Arabic, "multi_long", "إدارة الاتصالات بعدة مضيفين مع معالجة جلسات متقدمة")
	message.SetString(language.Bengali, "multi_long", "উন্নত সেশন হ্যান্ডলিং সহ একাধিক হোস্টে সংযোগ পরিচালনা")
	message.SetString(language.Portuguese, "multi_long", "Gerenciar conexões para múltiplos hosts com tratamento avançado de sessões")
	message.SetString(language.Russian, "multi_long", "Управление подключениями к нескольким хостам с расширенной обработкой сеансов")
	message.SetString(language.Japanese, "multi_long", "高度なセッション処理で複数ホストへの接続を管理")
	message.SetString(language.German, "multi_long", "Verbindungen zu mehreren Hosts mit erweiterte Sitzungsbehandlung verwalten")
	message.SetString(language.French, "multi_long", "Gérer les connexions à plusieurs hôtes avec gestion avancée de session")

	message.SetString(language.English, "multi_examples", `  # Connect to multiple hosts
  ts-ssh multi --hosts "host1,host2,host3"
  
  # Use tmux for session management
  ts-ssh multi --hosts "host1,host2" --tmux`)
	message.SetString(language.Spanish, "multi_examples", `  # Conectar a múltiples hosts
  ts-ssh multi --hosts "host1,host2,host3"
  
  # Usar tmux para gestión de sesiones
  ts-ssh multi --hosts "host1,host2" --tmux`)
	message.SetString(language.Chinese, "multi_examples", `  # 连接到多个主机
  ts-ssh multi --hosts "host1,host2,host3"
  
  # 使用 tmux 进行会话管理
  ts-ssh multi --hosts "host1,host2" --tmux`)
	message.SetString(language.German, "multi_examples", `  # Zu mehreren Hosts verbinden
  ts-ssh multi --hosts "host1,host2,host3"
  
  # Tmux für Sitzungsmanagement verwenden
  ts-ssh multi --hosts "host1,host2" --tmux`)
	message.SetString(language.French, "multi_examples", `  # Connecter à plusieurs hôtes
  ts-ssh multi --hosts "host1,host2,host3"
  
  # Utiliser tmux pour la gestion de session
  ts-ssh multi --hosts "host1,host2" --tmux`)

	// Config command
	message.SetString(language.English, "config_short", "Manage configuration")
	message.SetString(language.Spanish, "config_short", "Gestionar configuración")
	message.SetString(language.Chinese, "config_short", "管理配置")
	message.SetString(language.Hindi, "config_short", "कॉन्फ़िगरेशन प्रबंधित करें")
	message.SetString(language.Arabic, "config_short", "إدارة التكوين")
	message.SetString(language.Bengali, "config_short", "কনফিগারেশন পরিচালনা")
	message.SetString(language.Portuguese, "config_short", "Gerenciar configuração")
	message.SetString(language.Russian, "config_short", "Управление конфигурацией")
	message.SetString(language.Japanese, "config_short", "設定管理")
	message.SetString(language.German, "config_short", "Konfiguration verwalten")
	message.SetString(language.French, "config_short", "Gérer la configuration")

	message.SetString(language.English, "config_long", "View and modify ts-ssh configuration settings")
	message.SetString(language.Spanish, "config_long", "Ver y modificar configuraciones de ts-ssh")
	message.SetString(language.Chinese, "config_long", "查看和修改 ts-ssh 配置设置")
	message.SetString(language.Hindi, "config_long", "ts-ssh कॉन्फ़िगरेशन सेटिंग्स देखें और संशोधित करें")
	message.SetString(language.Arabic, "config_long", "عرض وتعديل إعدادات تكوين ts-ssh")
	message.SetString(language.Bengali, "config_long", "ts-ssh কনফিগারেশন সেটিংস দেখুন এবং পরিবর্তন করুন")
	message.SetString(language.Portuguese, "config_long", "Visualizar e modificar configurações do ts-ssh")
	message.SetString(language.Russian, "config_long", "Просмотр и изменение настроек конфигурации ts-ssh")
	message.SetString(language.Japanese, "config_long", "ts-ssh設定の表示と変更")
	message.SetString(language.German, "config_long", "ts-ssh Konfigurationseinstellungen anzeigen und ändern")
	message.SetString(language.French, "config_long", "Afficher et modifier les paramètres de configuration ts-ssh")

	message.SetString(language.English, "config_examples", `  # Show configuration
  ts-ssh config --show
  
  # Set a value
  ts-ssh config --set "user=myuser"
  
  # Reset to defaults
  ts-ssh config --reset`)
	message.SetString(language.Spanish, "config_examples", `  # Mostrar configuración
  ts-ssh config --show
  
  # Establecer un valor
  ts-ssh config --set "user=myuser"
  
  # Restaurar valores predeterminados
  ts-ssh config --reset`)
	message.SetString(language.Chinese, "config_examples", `  # 显示配置
  ts-ssh config --show
  
  # 设置值
  ts-ssh config --set "user=myuser"
  
  # 重置为默认值
  ts-ssh config --reset`)
	message.SetString(language.German, "config_examples", `  # Konfiguration anzeigen
  ts-ssh config --show
  
  # Einen Wert setzen
  ts-ssh config --set "user=myuser"
  
  # Auf Standardwerte zurücksetzen
  ts-ssh config --reset`)
	message.SetString(language.French, "config_examples", `  # Afficher la configuration
  ts-ssh config --show
  
  # Définir une valeur
  ts-ssh config --set "user=myuser"
  
  # Réinitialiser aux valeurs par défaut
  ts-ssh config --reset`)

	// PQC command
	message.SetString(language.English, "pqc_short", "Post-quantum cryptography operations")
	message.SetString(language.Spanish, "pqc_short", "Operaciones de criptografía post-cuántica")
	message.SetString(language.Chinese, "pqc_short", "后量子密码学操作")
	message.SetString(language.Hindi, "pqc_short", "पोस्ट-क्वांटम क्रिप्टोग्राफी ऑपरेशन्स")
	message.SetString(language.Arabic, "pqc_short", "عمليات التشفير ما بعد الكمومي")
	message.SetString(language.Bengali, "pqc_short", "পোস্ট-কোয়ান্টাম ক্রিপ্টোগ্রাফি অপারেশন")
	message.SetString(language.Portuguese, "pqc_short", "Operações de criptografia pós-quântica")
	message.SetString(language.Russian, "pqc_short", "Операции постквантовой криптографии")
	message.SetString(language.Japanese, "pqc_short", "ポスト量子暗号操作")
	message.SetString(language.German, "pqc_short", "Post-Quanten-Kryptographie-Operationen")
	message.SetString(language.French, "pqc_short", "Opérations de cryptographie post-quantique")

	message.SetString(language.English, "pqc_long", "Manage and test post-quantum cryptographic features")
	message.SetString(language.Spanish, "pqc_long", "Gestionar y probar características de criptografía post-cuántica")
	message.SetString(language.Chinese, "pqc_long", "管理和测试后量子密码学特性")
	message.SetString(language.Hindi, "pqc_long", "पोस्ट-क्वांटम क्रिप्टोग्राफिक फीचर्स का प्रबंधन और परीक्षण करें")
	message.SetString(language.Arabic, "pqc_long", "إدارة واختبار ميزات التشفير ما بعد الكمومي")
	message.SetString(language.Bengali, "pqc_long", "পোস্ট-কোয়ান্টাম ক্রিপ্টোগ্রাফিক বৈশিষ্ট্য পরিচালনা এবং পরীক্ষা")
	message.SetString(language.Portuguese, "pqc_long", "Gerenciar e testar recursos de criptografia pós-quântica")
	message.SetString(language.Russian, "pqc_long", "Управление и тестирование возможностей постквантовой криптографии")
	message.SetString(language.Japanese, "pqc_long", "ポスト量子暗号機能の管理とテスト")
	message.SetString(language.German, "pqc_long", "Post-Quanten-Kryptographie-Features verwalten und testen")
	message.SetString(language.French, "pqc_long", "Gérer et tester les fonctionnalités de cryptographie post-quantique")

	message.SetString(language.English, "pqc_examples", `  # Show PQC status
  ts-ssh pqc
  
  # Generate report
  ts-ssh pqc --report
  
  # Run benchmarks
  ts-ssh pqc --benchmark`)
	message.SetString(language.Spanish, "pqc_examples", `  # Mostrar estado PQC
  ts-ssh pqc
  
  # Generar informe
  ts-ssh pqc --report
  
  # Ejecutar benchmarks
  ts-ssh pqc --benchmark`)
	message.SetString(language.Chinese, "pqc_examples", `  # 显示 PQC 状态
  ts-ssh pqc
  
  # 生成报告
  ts-ssh pqc --report
  
  # 运行基准测试
  ts-ssh pqc --benchmark`)
	message.SetString(language.German, "pqc_examples", `  # PQC-Status anzeigen
  ts-ssh pqc
  
  # Bericht generieren
  ts-ssh pqc --report
  
  # Benchmarks ausführen
  ts-ssh pqc --benchmark`)
	message.SetString(language.French, "pqc_examples", `  # Afficher le statut PQC
  ts-ssh pqc
  
  # Générer un rapport
  ts-ssh pqc --report
  
  # Exécuter des benchmarks
  ts-ssh pqc --benchmark`)

	// Version command
	message.SetString(language.English, "version_short", "Show version information")
	message.SetString(language.Spanish, "version_short", "Mostrar información de versión")
	message.SetString(language.Chinese, "version_short", "显示版本信息")
	message.SetString(language.Hindi, "version_short", "संस्करण जानकारी दिखाएं")
	message.SetString(language.Arabic, "version_short", "عرض معلومات الإصدار")
	message.SetString(language.Bengali, "version_short", "সংস্করণ তথ্য দেখান")
	message.SetString(language.Portuguese, "version_short", "Mostrar informações da versão")
	message.SetString(language.Russian, "version_short", "Показать информацию о версии")
	message.SetString(language.Japanese, "version_short", "バージョン情報を表示")
	message.SetString(language.German, "version_short", "Versionsinformationen anzeigen")
	message.SetString(language.French, "version_short", "Afficher les informations de version")

	message.SetString(language.English, "version_long", "Display version and build information for ts-ssh")
	message.SetString(language.Spanish, "version_long", "Mostrar información de versión y compilación para ts-ssh")
	message.SetString(language.Chinese, "version_long", "显示 ts-ssh 的版本和构建信息")
	message.SetString(language.Hindi, "version_long", "ts-ssh के लिए संस्करण और बिल्ड जानकारी प्रदर्शित करें")
	message.SetString(language.Arabic, "version_long", "عرض معلومات الإصدار والبناء لـ ts-ssh")
	message.SetString(language.Bengali, "version_long", "ts-ssh এর জন্য সংস্করণ এবং বিল্ড তথ্য প্রদর্শন")
	message.SetString(language.Portuguese, "version_long", "Exibir informações de versão e build para ts-ssh")
	message.SetString(language.Russian, "version_long", "Отобразить информацию о версии и сборке для ts-ssh")
	message.SetString(language.Japanese, "version_long", "ts-sshのバージョンとビルド情報を表示")
	message.SetString(language.German, "version_long", "Versions- und Build-Informationen für ts-ssh anzeigen")
	message.SetString(language.French, "version_long", "Afficher les informations de version et de build pour ts-ssh")

	// Flag descriptions
	message.SetString(language.English, "flag_user_help", "SSH username for connection")
	message.SetString(language.Spanish, "flag_user_help", "Nombre de usuario SSH para la conexión")
	message.SetString(language.Chinese, "flag_user_help", "连接用的 SSH 用户名")
	message.SetString(language.Hindi, "flag_user_help", "कनेक्शन के लिए SSH यूज़रनेम")
	message.SetString(language.Arabic, "flag_user_help", "اسم مستخدم SSH للاتصال")
	message.SetString(language.Bengali, "flag_user_help", "কনেকশনের জন্য SSH ব্যবহারকারীর নাম")
	message.SetString(language.Portuguese, "flag_user_help", "Nome de usuário SSH para conexão")
	message.SetString(language.Russian, "flag_user_help", "Имя пользователя SSH для подключения")
	message.SetString(language.Japanese, "flag_user_help", "接続用SSHユーザー名")
	message.SetString(language.German, "flag_user_help", "SSH-Benutzername für Verbindung")
	message.SetString(language.French, "flag_user_help", "Nom d'utilisateur SSH pour la connexion")

	message.SetString(language.English, "flag_identity_help", "Path to SSH private key file")
	message.SetString(language.Spanish, "flag_identity_help", "Ruta al archivo de clave privada SSH")
	message.SetString(language.Chinese, "flag_identity_help", "SSH 私钥文件路径")
	message.SetString(language.Hindi, "flag_identity_help", "SSH प्राइवेट की फाइल का पाथ")
	message.SetString(language.Arabic, "flag_identity_help", "مسار ملف مفتاح SSH الخاص")
	message.SetString(language.Bengali, "flag_identity_help", "SSH প্রাইভেট কী ফাইলের পথ")
	message.SetString(language.Portuguese, "flag_identity_help", "Caminho para arquivo de chave privada SSH")
	message.SetString(language.Russian, "flag_identity_help", "Путь к файлу частного ключа SSH")
	message.SetString(language.Japanese, "flag_identity_help", "SSH秘密鍵ファイルのパス")
	message.SetString(language.German, "flag_identity_help", "Pfad zur SSH-Private-Key-Datei")
	message.SetString(language.French, "flag_identity_help", "Chemin vers le fichier de clé privée SSH")

	message.SetString(language.English, "flag_lang_help", "Set language for output (en, es, fr, de, etc.)")
	message.SetString(language.Spanish, "flag_lang_help", "Establecer idioma para la salida (en, es, fr, de, etc.)")
	message.SetString(language.Chinese, "flag_lang_help", "设置输出语言 (en, es, fr, de, 等)")
	message.SetString(language.Hindi, "flag_lang_help", "आउटपुट के लिए भाषा सेट करें (en, es, fr, de, आदि)")
	message.SetString(language.Arabic, "flag_lang_help", "تعيين لغة الإخراج (en, es, fr, de, إلخ)")
	message.SetString(language.Bengali, "flag_lang_help", "আউটপুটের জন্য ভাষা সেট করুন (en, es, fr, de, ইত্যাদি)")
	message.SetString(language.Portuguese, "flag_lang_help", "Definir idioma para saída (en, es, fr, de, etc.)")
	message.SetString(language.Russian, "flag_lang_help", "Установить язык вывода (en, es, fr, de, и т.д.)")
	message.SetString(language.Japanese, "flag_lang_help", "出力言語を設定 (en, es, fr, de, など)")
	message.SetString(language.German, "flag_lang_help", "Sprache für Ausgabe festlegen (en, es, fr, de, usw.)")
	message.SetString(language.French, "flag_lang_help", "Définir la langue de sortie (en, es, fr, de, etc.)")

	// Additional flag descriptions
	message.SetString(language.English, "flag_config_help", "SSH config file path")
	message.SetString(language.Spanish, "flag_config_help", "Ruta del archivo de configuración SSH")
	message.SetString(language.Chinese, "flag_config_help", "SSH 配置文件路径")
	message.SetString(language.German, "flag_config_help", "SSH-Konfigurationsdateipfad")
	message.SetString(language.French, "flag_config_help", "Chemin du fichier de configuration SSH")

	message.SetString(language.English, "flag_tsnet_help", "Directory for tsnet state and logs")
	message.SetString(language.Spanish, "flag_tsnet_help", "Directorio para estado y logs de tsnet")
	message.SetString(language.Chinese, "flag_tsnet_help", "tsnet 状态和日志目录")
	message.SetString(language.German, "flag_tsnet_help", "Verzeichnis für tsnet-Status und -Logs")
	message.SetString(language.French, "flag_tsnet_help", "Répertoire pour l'état et les journaux tsnet")

	message.SetString(language.English, "flag_control_help", "Tailscale control server URL")
	message.SetString(language.Spanish, "flag_control_help", "URL del servidor de control Tailscale")
	message.SetString(language.Chinese, "flag_control_help", "Tailscale 控制服务器 URL")
	message.SetString(language.German, "flag_control_help", "Tailscale-Kontrollserver-URL")
	message.SetString(language.French, "flag_control_help", "URL du serveur de contrôle Tailscale")

	message.SetString(language.English, "flag_verbose_help", "Enable verbose logging")
	message.SetString(language.Spanish, "flag_verbose_help", "Habilitar logging detallado")
	message.SetString(language.Chinese, "flag_verbose_help", "启用详细日志")
	message.SetString(language.German, "flag_verbose_help", "Ausführliche Protokollierung aktivieren")
	message.SetString(language.French, "flag_verbose_help", "Activer la journalisation détaillée")

	message.SetString(language.English, "flag_insecure_help", "Skip host key verification (insecure)")
	message.SetString(language.Spanish, "flag_insecure_help", "Omitir verificación de clave del servidor (inseguro)")
	message.SetString(language.Chinese, "flag_insecure_help", "跳过主机密钥验证（不安全）")
	message.SetString(language.German, "flag_insecure_help", "Host-Schlüssel-Verifikation überspringen (unsicher)")
	message.SetString(language.French, "flag_insecure_help", "Ignorer la vérification de clé d'hôte (non sécurisé)")

	message.SetString(language.English, "flag_force_insecure_help", "Force insecure mode without confirmation")
	message.SetString(language.Spanish, "flag_force_insecure_help", "Forzar modo inseguro sin confirmación")
	message.SetString(language.Chinese, "flag_force_insecure_help", "强制不安全模式而不确认")
	message.SetString(language.German, "flag_force_insecure_help", "Unsicheren Modus ohne Bestätigung erzwingen")
	message.SetString(language.French, "flag_force_insecure_help", "Forcer le mode non sécurisé sans confirmation")

	message.SetString(language.English, "flag_pqc_help", "Enable post-quantum cryptography")
	message.SetString(language.Spanish, "flag_pqc_help", "Habilitar criptografía post-cuántica")
	message.SetString(language.Chinese, "flag_pqc_help", "启用后量子密码学")
	message.SetString(language.German, "flag_pqc_help", "Post-Quanten-Kryptographie aktivieren")
	message.SetString(language.French, "flag_pqc_help", "Activer la cryptographie post-quantique")

	message.SetString(language.English, "flag_pqc_level_help", "PQC level: 0=none, 1=hybrid, 2=strict")
	message.SetString(language.Spanish, "flag_pqc_level_help", "Nivel PQC: 0=ninguno, 1=híbrido, 2=estricto")
	message.SetString(language.Chinese, "flag_pqc_level_help", "PQC 级别: 0=无, 1=混合, 2=严格")
	message.SetString(language.German, "flag_pqc_level_help", "PQC-Level: 0=keine, 1=hybrid, 2=strikt")
	message.SetString(language.French, "flag_pqc_level_help", "Niveau PQC: 0=aucun, 1=hybride, 2=strict")
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

// detectLanguageFromArgs parses command line arguments early to detect --lang flag
// This allows us to initialize i18n with the correct language before creating Cobra commands
func detectLanguageFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "--lang" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--lang=") {
			return strings.TrimPrefix(arg, "--lang=")
		}
	}
	return "" // Default language will be determined by initI18n
}

// initI18nForCLI initializes i18n early for Cobra CLI with language detection from args
func initI18nForCLI(args []string) {
	lang := detectLanguageFromArgs(args)
	initI18n(lang)
}