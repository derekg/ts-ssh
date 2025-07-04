package i18n

import (
	"os"
	"strings"
	"sync"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Re-export supported languages from main package
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

// InitI18n initializes the internationalization system thread-safely
func InitI18n(langFlag string) {
	// Ensure messages are registered only once across all goroutines
	initI18nOnce.Do(func() {
		registerInternalMessages()
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

// T returns a localized string using the global printer thread-safely
func T(key string, args ...interface{}) string {
	// Read printer with read lock for concurrent access
	printerMu.RLock()
	p := printer
	printerMu.RUnlock()

	// Initialize if not yet done
	if p == nil {
		InitI18n("")
		printerMu.RLock()
		p = printer
		printerMu.RUnlock()
	}

	// Use local copy to avoid holding lock during sprintf
	return p.Sprintf(key, args...)
}

// registerInternalMessages registers translatable messages used by internal packages
func registerInternalMessages() {
	// Security TTY messages
	message.SetString(language.English, "tty_path_validation_failed", "TTY path validation failed")
	message.SetString(language.Spanish, "tty_path_validation_failed", "Error en la validación de la ruta TTY")
	message.SetString(language.Chinese, "tty_path_validation_failed", "TTY路径验证失败")
	message.SetString(language.Hindi, "tty_path_validation_failed", "TTY पथ सत्यापन विफल")
	message.SetString(language.Arabic, "tty_path_validation_failed", "فشل التحقق من مسار TTY")
	message.SetString(language.Bengali, "tty_path_validation_failed", "TTY পথ যাচাইকরণ ব্যর্থ")
	message.SetString(language.Portuguese, "tty_path_validation_failed", "Falha na validação do caminho TTY")
	message.SetString(language.Russian, "tty_path_validation_failed", "Ошибка проверки пути TTY")
	message.SetString(language.Japanese, "tty_path_validation_failed", "TTYパスの検証に失敗しました")
	message.SetString(language.German, "tty_path_validation_failed", "TTY-Pfad-Validierung fehlgeschlagen")
	message.SetString(language.French, "tty_path_validation_failed", "échec de validation du chemin TTY")

	message.SetString(language.English, "tty_ownership_check_failed", "TTY ownership check failed")
	message.SetString(language.Spanish, "tty_ownership_check_failed", "Error en la verificación de propiedad TTY")
	message.SetString(language.Chinese, "tty_ownership_check_failed", "TTY所有权检查失败")
	message.SetString(language.Hindi, "tty_ownership_check_failed", "TTY स्वामित्व जाँच विफल")
	message.SetString(language.Arabic, "tty_ownership_check_failed", "فشل فحص ملكية TTY")
	message.SetString(language.Bengali, "tty_ownership_check_failed", "TTY মালিকানা পরীক্ষা ব্যর্থ")
	message.SetString(language.Portuguese, "tty_ownership_check_failed", "Falha na verificação de propriedade TTY")
	message.SetString(language.Russian, "tty_ownership_check_failed", "Ошибка проверки владения TTY")
	message.SetString(language.Japanese, "tty_ownership_check_failed", "TTY所有権チェックに失敗しました")
	message.SetString(language.German, "tty_ownership_check_failed", "TTY-Eigentümerschaftsprüfung fehlgeschlagen")
	message.SetString(language.French, "tty_ownership_check_failed", "échec de vérification de propriété TTY")

	message.SetString(language.English, "tty_permission_check_failed", "TTY permission check failed")
	message.SetString(language.Spanish, "tty_permission_check_failed", "Error en la verificación de permisos TTY")
	message.SetString(language.Chinese, "tty_permission_check_failed", "TTY权限检查失败")
	message.SetString(language.Hindi, "tty_permission_check_failed", "TTY अनुमति जाँच विफल")
	message.SetString(language.Arabic, "tty_permission_check_failed", "فشل فحص صلاحية TTY")
	message.SetString(language.Bengali, "tty_permission_check_failed", "TTY অনুমতি পরীক্ষা ব্যর্থ")
	message.SetString(language.Portuguese, "tty_permission_check_failed", "Falha na verificação de permissão TTY")
	message.SetString(language.Russian, "tty_permission_check_failed", "Ошибка проверки разрешений TTY")
	message.SetString(language.Japanese, "tty_permission_check_failed", "TTY権限チェックに失敗しました")
	message.SetString(language.German, "tty_permission_check_failed", "TTY-Berechtigungsprüfung fehlgeschlagen")
	message.SetString(language.French, "tty_permission_check_failed", "échec de vérification des permissions TTY")

	message.SetString(language.English, "not_running_in_terminal", "not running in terminal")
	message.SetString(language.Spanish, "not_running_in_terminal", "no se ejecuta en terminal")
	message.SetString(language.Chinese, "not_running_in_terminal", "未在终端中运行")
	message.SetString(language.Hindi, "not_running_in_terminal", "टर्मिनल में नहीं चल रहा")
	message.SetString(language.Arabic, "not_running_in_terminal", "لا يعمل في المحطة الطرفية")
	message.SetString(language.Bengali, "not_running_in_terminal", "টার্মিনালে চলছে না")
	message.SetString(language.Portuguese, "not_running_in_terminal", "não está executando no terminal")
	message.SetString(language.Russian, "not_running_in_terminal", "не работает в терминале")
	message.SetString(language.Japanese, "not_running_in_terminal", "ターミナルで実行されていません")
	message.SetString(language.German, "not_running_in_terminal", "läuft nicht im Terminal")
	message.SetString(language.French, "not_running_in_terminal", "ne fonctionne pas dans le terminal")

	message.SetString(language.English, "tty_security_validation_failed", "TTY security validation failed")
	message.SetString(language.Spanish, "tty_security_validation_failed", "Error en la validación de seguridad TTY")
	message.SetString(language.Chinese, "tty_security_validation_failed", "TTY安全验证失败")
	message.SetString(language.Hindi, "tty_security_validation_failed", "TTY सुरक्षा सत्यापन विफल")
	message.SetString(language.Arabic, "tty_security_validation_failed", "فشل التحقق من أمان TTY")
	message.SetString(language.Bengali, "tty_security_validation_failed", "TTY নিরাপত্তা যাচাইকরণ ব্যর্থ")
	message.SetString(language.Portuguese, "tty_security_validation_failed", "Falha na validação de segurança TTY")
	message.SetString(language.Russian, "tty_security_validation_failed", "Ошибка проверки безопасности TTY")
	message.SetString(language.Japanese, "tty_security_validation_failed", "TTYセキュリティ検証に失敗しました")
	message.SetString(language.German, "tty_security_validation_failed", "TTY-Sicherheitsvalidierung fehlgeschlagen")
	message.SetString(language.French, "tty_security_validation_failed", "échec de validation de sécurité TTY")

	message.SetString(language.English, "failed_open_tty", "failed to open TTY")
	message.SetString(language.Spanish, "failed_open_tty", "error al abrir TTY")
	message.SetString(language.Chinese, "failed_open_tty", "无法打开TTY")
	message.SetString(language.Hindi, "failed_open_tty", "TTY खोलने में विफल")
	message.SetString(language.Arabic, "failed_open_tty", "فشل فتح TTY")
	message.SetString(language.Bengali, "failed_open_tty", "TTY খুলতে ব্যর্থ")
	message.SetString(language.Portuguese, "failed_open_tty", "falha ao abrir TTY")
	message.SetString(language.Russian, "failed_open_tty", "не удалось открыть TTY")
	message.SetString(language.Japanese, "failed_open_tty", "TTYを開くことができませんでした")
	message.SetString(language.German, "failed_open_tty", "TTY konnte nicht geöffnet werden")
	message.SetString(language.French, "failed_open_tty", "échec d'ouverture TTY")

	// SSH connection messages
	message.SetString(language.English, "host_key_warning", "WARNING: Host key verification is disabled")
	message.SetString(language.Spanish, "host_key_warning", "ADVERTENCIA: La verificación de clave de host está deshabilitada")
	message.SetString(language.Chinese, "host_key_warning", "警告：主机密钥验证已禁用")
	message.SetString(language.Hindi, "host_key_warning", "चेतावनी: होस्ट की सत्यापन अक्षम है")
	message.SetString(language.Arabic, "host_key_warning", "تحذير: تحقق مفتاح المضيف معطل")
	message.SetString(language.Bengali, "host_key_warning", "সতর্কতা: হোস্ট কী যাচাইকরণ অক্ষম")
	message.SetString(language.Portuguese, "host_key_warning", "AVISO: A verificação da chave do host está desabilitada")
	message.SetString(language.Russian, "host_key_warning", "ПРЕДУПРЕЖДЕНИЕ: Проверка ключа хоста отключена")
	message.SetString(language.Japanese, "host_key_warning", "警告：ホストキーの検証が無効になっています")
	message.SetString(language.German, "host_key_warning", "WARNUNG: Host-Schlüssel-Überprüfung ist deaktiviert")
	message.SetString(language.French, "host_key_warning", "AVERTISSEMENT : La vérification de la clé d'hôte est désactivée")

	message.SetString(language.English, "dial_via_tsnet", "Connecting via tsnet...")
	message.SetString(language.Spanish, "dial_via_tsnet", "Conectando vía tsnet...")
	message.SetString(language.Chinese, "dial_via_tsnet", "通过tsnet连接中...")
	message.SetString(language.Hindi, "dial_via_tsnet", "tsnet के माध्यम से कनेक्ट हो रहा है...")
	message.SetString(language.Arabic, "dial_via_tsnet", "الاتصال عبر tsnet...")
	message.SetString(language.Bengali, "dial_via_tsnet", "tsnet এর মাধ্যমে সংযোগ করা হচ্ছে...")
	message.SetString(language.Portuguese, "dial_via_tsnet", "Conectando via tsnet...")
	message.SetString(language.Russian, "dial_via_tsnet", "Подключение через tsnet...")
	message.SetString(language.Japanese, "dial_via_tsnet", "tsnet経由で接続中...")
	message.SetString(language.German, "dial_via_tsnet", "Verbindung über tsnet...")
	message.SetString(language.French, "dial_via_tsnet", "Connexion via tsnet...")

	message.SetString(language.English, "ssh_handshake", "Performing SSH handshake...")
	message.SetString(language.Spanish, "ssh_handshake", "Realizando protocolo SSH...")
	message.SetString(language.Chinese, "ssh_handshake", "正在执行SSH握手...")
	message.SetString(language.Hindi, "ssh_handshake", "SSH हैंडशेक कर रहा है...")
	message.SetString(language.Arabic, "ssh_handshake", "إجراء مصافحة SSH...")
	message.SetString(language.Bengali, "ssh_handshake", "SSH হ্যান্ডশেক সম্পাদন করা হচ্ছে...")
	message.SetString(language.Portuguese, "ssh_handshake", "Realizando handshake SSH...")
	message.SetString(language.Russian, "ssh_handshake", "Выполнение рукопожатия SSH...")
	message.SetString(language.Japanese, "ssh_handshake", "SSHハンドシェイクを実行中...")
	message.SetString(language.German, "ssh_handshake", "SSH-Handshake wird durchgeführt...")
	message.SetString(language.French, "ssh_handshake", "Exécution de la poignée de main SSH...")

	message.SetString(language.English, "dial_failed", "connection failed")
	message.SetString(language.Spanish, "dial_failed", "conexión falló")
	message.SetString(language.Chinese, "dial_failed", "连接失败")
	message.SetString(language.Hindi, "dial_failed", "कनेक्शन विफल")
	message.SetString(language.Arabic, "dial_failed", "فشل الاتصال")
	message.SetString(language.Bengali, "dial_failed", "সংযোগ ব্যর্থ")
	message.SetString(language.Portuguese, "dial_failed", "conexão falhou")
	message.SetString(language.Russian, "dial_failed", "соединение не удалось")
	message.SetString(language.Japanese, "dial_failed", "接続に失敗しました")
	message.SetString(language.German, "dial_failed", "Verbindung fehlgeschlagen")
	message.SetString(language.French, "dial_failed", "échec de connexion")

	message.SetString(language.English, "ssh_connection_failed", "SSH connection failed")
	message.SetString(language.Spanish, "ssh_connection_failed", "Conexión SSH falló")
	message.SetString(language.Chinese, "ssh_connection_failed", "SSH连接失败")
	message.SetString(language.Hindi, "ssh_connection_failed", "SSH कनेक्शन विफल")
	message.SetString(language.Arabic, "ssh_connection_failed", "فشل اتصال SSH")
	message.SetString(language.Bengali, "ssh_connection_failed", "SSH সংযোগ ব্যর্থ")
	message.SetString(language.Portuguese, "ssh_connection_failed", "Conexão SSH falhou")
	message.SetString(language.Russian, "ssh_connection_failed", "SSH соединение не удалось")
	message.SetString(language.Japanese, "ssh_connection_failed", "SSH接続に失敗しました")
	message.SetString(language.German, "ssh_connection_failed", "SSH-Verbindung fehlgeschlagen")
	message.SetString(language.French, "ssh_connection_failed", "échec de connexion SSH")

	message.SetString(language.English, "ssh_connection_established", "SSH connection established")
	message.SetString(language.Spanish, "ssh_connection_established", "Conexión SSH establecida")
	message.SetString(language.Chinese, "ssh_connection_established", "SSH连接已建立")
	message.SetString(language.Hindi, "ssh_connection_established", "SSH कनेक्शन स्थापित")
	message.SetString(language.Arabic, "ssh_connection_established", "تم تأسيس اتصال SSH")
	message.SetString(language.Bengali, "ssh_connection_established", "SSH সংযোগ প্রতিষ্ঠিত")
	message.SetString(language.Portuguese, "ssh_connection_established", "Conexão SSH estabelecida")
	message.SetString(language.Russian, "ssh_connection_established", "SSH соединение установлено")
	message.SetString(language.Japanese, "ssh_connection_established", "SSH接続が確立されました")
	message.SetString(language.German, "ssh_connection_established", "SSH-Verbindung hergestellt")
	message.SetString(language.French, "ssh_connection_established", "Connexion SSH établie")

	// SCP operation messages
	message.SetString(language.English, "scp_empty_path", "SCP path cannot be empty")
	message.SetString(language.Spanish, "scp_empty_path", "La ruta SCP no puede estar vacía")
	message.SetString(language.Chinese, "scp_empty_path", "SCP路径不能为空")
	message.SetString(language.Hindi, "scp_empty_path", "SCP पथ खाली नहीं हो सकता")
	message.SetString(language.Arabic, "scp_empty_path", "مسار SCP لا يمكن أن يكون فارغاً")
	message.SetString(language.Bengali, "scp_empty_path", "SCP পথ খালি থাকতে পারে না")
	message.SetString(language.Portuguese, "scp_empty_path", "O caminho SCP não pode estar vazio")
	message.SetString(language.Russian, "scp_empty_path", "Путь SCP не может быть пустым")
	message.SetString(language.Japanese, "scp_empty_path", "SCPパスは空にできません")
	message.SetString(language.German, "scp_empty_path", "SCP-Pfad darf nicht leer sein")
	message.SetString(language.French, "scp_empty_path", "Le chemin SCP ne peut pas être vide")

	message.SetString(language.English, "scp_enter_password", "Enter password for %s@%s: ")
	message.SetString(language.Spanish, "scp_enter_password", "Ingrese contraseña para %s@%s: ")
	message.SetString(language.Chinese, "scp_enter_password", "为 %s@%s 输入密码: ")
	message.SetString(language.Hindi, "scp_enter_password", "%s@%s के लिए पासवर्ड दर्ज करें: ")
	message.SetString(language.Arabic, "scp_enter_password", "أدخل كلمة المرور لـ %s@%s: ")
	message.SetString(language.Bengali, "scp_enter_password", "%s@%s এর জন্য পাসওয়ার্ড লিখুন: ")
	message.SetString(language.Portuguese, "scp_enter_password", "Digite a senha para %s@%s: ")
	message.SetString(language.Russian, "scp_enter_password", "Введите пароль для %s@%s: ")
	message.SetString(language.Japanese, "scp_enter_password", "%s@%s のパスワードを入力してください: ")
	message.SetString(language.German, "scp_enter_password", "Passwort für %s@%s eingeben: ")
	message.SetString(language.French, "scp_enter_password", "Entrez le mot de passe pour %s@%s: ")

	message.SetString(language.English, "scp_host_key_warning", "WARNING: SCP host key verification disabled")
	message.SetString(language.Spanish, "scp_host_key_warning", "ADVERTENCIA: Verificación de clave de host SCP deshabilitada")
	message.SetString(language.Chinese, "scp_host_key_warning", "警告：SCP主机密钥验证已禁用")
	message.SetString(language.Hindi, "scp_host_key_warning", "चेतावनी: SCP होस्ट की सत्यापन अक्षम")
	message.SetString(language.Arabic, "scp_host_key_warning", "تحذير: تحقق مفتاح مضيف SCP معطل")
	message.SetString(language.Bengali, "scp_host_key_warning", "সতর্কতা: SCP হোস্ট কী যাচাইকরণ অক্ষম")
	message.SetString(language.Portuguese, "scp_host_key_warning", "AVISO: Verificação de chave de host SCP desabilitada")
	message.SetString(language.Russian, "scp_host_key_warning", "ПРЕДУПРЕЖДЕНИЕ: Проверка ключа хоста SCP отключена")
	message.SetString(language.Japanese, "scp_host_key_warning", "警告：SCPホストキーの検証が無効になっています")
	message.SetString(language.German, "scp_host_key_warning", "WARNUNG: SCP-Host-Schlüssel-Überprüfung ist deaktiviert")
	message.SetString(language.French, "scp_host_key_warning", "AVERTISSEMENT : La vérification de la clé d'hôte SCP est désactivée")

	message.SetString(language.English, "scp_upload_complete", "Upload complete")
	message.SetString(language.Spanish, "scp_upload_complete", "Carga completada")
	message.SetString(language.Chinese, "scp_upload_complete", "上传完成")
	message.SetString(language.Hindi, "scp_upload_complete", "अपलोड पूर्ण")
	message.SetString(language.Arabic, "scp_upload_complete", "اكتمل الرفع")
	message.SetString(language.Bengali, "scp_upload_complete", "আপলোড সম্পন্ন")
	message.SetString(language.Portuguese, "scp_upload_complete", "Upload concluído")
	message.SetString(language.Russian, "scp_upload_complete", "Загрузка завершена")
	message.SetString(language.Japanese, "scp_upload_complete", "アップロード完了")
	message.SetString(language.German, "scp_upload_complete", "Upload abgeschlossen")
	message.SetString(language.French, "scp_upload_complete", "Téléchargement terminé")

	message.SetString(language.English, "scp_download_complete", "Download complete")
	message.SetString(language.Spanish, "scp_download_complete", "Descarga completada")
	message.SetString(language.Chinese, "scp_download_complete", "下载完成")
	message.SetString(language.Hindi, "scp_download_complete", "डाउनलोड पूर्ण")
	message.SetString(language.Arabic, "scp_download_complete", "اكتمل التنزيل")
	message.SetString(language.Bengali, "scp_download_complete", "ডাউনলোড সম্পন্ন")
	message.SetString(language.Portuguese, "scp_download_complete", "Download concluído")
	message.SetString(language.Russian, "scp_download_complete", "Загрузка завершена")
	message.SetString(language.Japanese, "scp_download_complete", "ダウンロード完了")
	message.SetString(language.German, "scp_download_complete", "Download abgeschlossen")
	message.SetString(language.French, "scp_download_complete", "Téléchargement terminé")
}
