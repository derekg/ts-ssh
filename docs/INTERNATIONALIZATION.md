# ts-ssh Internationalization (i18n) Guide

This document describes the comprehensive internationalization support in ts-ssh, covering the top 11 most popular languages by speakers worldwide.

## Supported Languages

ts-ssh now supports **11 languages** covering over 4 billion native speakers globally:

| Language | Code | Script | Native Name | Speakers |
|----------|------|--------|-------------|----------|
| English | `en` | Latin | English | 380M |
| Spanish | `es` | Latin | Español | 460M |
| Chinese | `zh` | Simplified | 中文 | 918M |
| Hindi | `hi` | Devanagari | हिन्दी | 341M |
| Arabic | `ar` | Arabic | العربية | 422M |
| Bengali | `bn` | Bengali | বাংলা | 265M |
| Portuguese | `pt` | Latin | Português | 260M |
| Russian | `ru` | Cyrillic | Русский | 154M |
| Japanese | `ja` | Kanji/Hiragana | 日本語 | 125M |
| German | `de` | Latin | Deutsch | 95M |
| French | `fr` | Latin | Français | 80M |

## Language Detection

### Priority Order
1. **CLI flag**: `--lang <code>`
2. **Environment variable**: `TS_SSH_LANG=<code>`
3. **System locale**: `LC_ALL` environment variable
4. **System locale**: `LANG` environment variable
5. **Default**: English (`en`)

### Language Code Variations

Each language supports multiple input formats:

#### English (`en`)
- `en`, `english`, `en_us`, `en-us`, `en_gb`, `en-gb`

#### Spanish (`es`)
- `es`, `spanish`, `español`, `es_es`, `es-es`, `es_mx`, `es-mx`

#### Chinese (`zh`)
- `zh`, `chinese`, `中文`, `zh-cn`, `zh-tw`, `zh_cn`, `zh_tw`

#### Hindi (`hi`)
- `hi`, `hindi`, `हिन्दी`, `hi_in`, `hi-in`

#### Arabic (`ar`)
- `ar`, `arabic`, `العربية`, `ar_sa`, `ar-sa`, `ar_eg`, `ar-eg`

#### Bengali (`bn`)
- `bn`, `bengali`, `বাংলা`, `bn_bd`, `bn-bd`, `bn_in`, `bn-in`

#### Portuguese (`pt`)
- `pt`, `portuguese`, `português`, `pt_br`, `pt-br`, `pt_pt`, `pt-pt`

#### Russian (`ru`)
- `ru`, `russian`, `русский`, `ru_ru`, `ru-ru`

#### Japanese (`ja`)
- `ja`, `japanese`, `日本語`, `ja_jp`, `ja-jp`

#### German (`de`)
- `de`, `german`, `deutsch`, `de_de`, `de-de`, `de_at`, `de-at`

#### French (`fr`)
- `fr`, `french`, `français`, `fr_fr`, `fr-fr`, `fr_ca`, `fr-ca`

## Usage Examples

### CLI Flag
```bash
# Use specific language
ts-ssh --lang=zh list              # Chinese
ts-ssh --lang=de connect host      # German
ts-ssh --lang=fr copy file host:/  # French
ts-ssh --lang=ar --help           # Arabic
ts-ssh --lang=ja version          # Japanese
```

### Environment Variables
```bash
# Set for session
export TS_SSH_LANG=pt
ts-ssh list                       # Portuguese

# Set for single command
TS_SSH_LANG=ru ts-ssh --help      # Russian

# System locale integration
export LANG=hi_IN.UTF-8
ts-ssh connect host               # Hindi

export LC_ALL=bn_BD.UTF-8
ts-ssh list                       # Bengali
```

### Legacy CLI Mode
```bash
# Force legacy mode with different languages
export TS_SSH_LEGACY_CLI=1

ts-ssh --lang=zh --list           # Chinese legacy interface
ts-ssh --lang=de --help           # German legacy interface
ts-ssh --lang=ar --version        # Arabic legacy interface
```

## Script Integration

### DevOps Scripts
```bash
#!/bin/bash
# Multi-language deployment script

case "$DEPLOY_REGION" in
  "cn")
    export TS_SSH_LANG=zh
    ;;
  "in")
    export TS_SSH_LANG=hi
    ;;
  "br")
    export TS_SSH_LANG=pt
    ;;
  "de")
    export TS_SSH_LANG=de
    ;;
  "fr")
    export TS_SSH_LANG=fr
    ;;
  "ru")
    export TS_SSH_LANG=ru
    ;;
  "jp")
    export TS_SSH_LANG=ja
    ;;
  "ar")
    export TS_SSH_LANG=ar
    ;;
  "bd")
    export TS_SSH_LANG=bn
    ;;
  *)
    export TS_SSH_LANG=en
    ;;
esac

ts-ssh exec --command "deploy.sh" web1,web2,web3
```

### CI/CD Integration
```yaml
# GitHub Actions
name: Deploy with Localization
jobs:
  deploy:
    strategy:
      matrix:
        region: [us, es, cn, in, de, fr, br, ru, jp, ar, bd]
    steps:
    - name: Set Language
      run: |
        case "${{ matrix.region }}" in
          "es") echo "TS_SSH_LANG=es" >> $GITHUB_ENV ;;
          "cn") echo "TS_SSH_LANG=zh" >> $GITHUB_ENV ;;
          "in") echo "TS_SSH_LANG=hi" >> $GITHUB_ENV ;;
          "de") echo "TS_SSH_LANG=de" >> $GITHUB_ENV ;;
          "fr") echo "TS_SSH_LANG=fr" >> $GITHUB_ENV ;;
          "br") echo "TS_SSH_LANG=pt" >> $GITHUB_ENV ;;
          "ru") echo "TS_SSH_LANG=ru" >> $GITHUB_ENV ;;
          "jp") echo "TS_SSH_LANG=ja" >> $GITHUB_ENV ;;
          "ar") echo "TS_SSH_LANG=ar" >> $GITHUB_ENV ;;
          "bd") echo "TS_SSH_LANG=bn" >> $GITHUB_ENV ;;
          *) echo "TS_SSH_LANG=en" >> $GITHUB_ENV ;;
        esac
    - name: Deploy
      run: ts-ssh exec --command "deploy.sh" ${{ matrix.region }}-servers
```

## Unicode and Encoding

### Character Support
- **UTF-8 encoding** for all languages
- **Unicode normalization** for consistent display
- **Right-to-left (RTL)** script support for Arabic
- **Complex scripts** support for Hindi (Devanagari), Bengali, and Japanese

### Terminal Compatibility
- Compatible with modern terminals supporting Unicode
- Proper rendering in Terminal.app, iTerm2, Windows Terminal, GNOME Terminal
- Fallback to ASCII for legacy terminals

## Translation Coverage

### Core Messages
All languages include translations for:
- Error messages and diagnostics
- Command descriptions and help text
- Status indicators (online/offline)
- Authentication prompts
- File operation messages
- Host discovery information
- Connection status updates

### Key Translation Examples

#### "No Tailscale peers found"
- **English**: No Tailscale peers found
- **Spanish**: No se encontraron pares Tailscale
- **Chinese**: 未找到 Tailscale 对等节点
- **Hindi**: कोई Tailscale पीयर नहीं मिला
- **Arabic**: لم يتم العثور على أقران Tailscale
- **Bengali**: কোন Tailscale পিয়ার পাওয়া যায়নি
- **Portuguese**: Nenhum par Tailscale encontrado
- **Russian**: Узлы Tailscale не найдены
- **Japanese**: Tailscaleピアが見つかりません
- **German**: Keine Tailscale-Peers gefunden
- **French**: Aucun pair Tailscale trouvé

#### "Failed to initialize Tailscale connection"
- **Chinese**: 初始化 Tailscale 连接失败
- **Hindi**: Tailscale कनेक्शन प्रारंभ करने में विफल
- **Arabic**: فشل في تهيئة اتصال Tailscale
- **Portuguese**: Falha ao inicializar conexão Tailscale
- **Russian**: Не удалось инициализировать соединение Tailscale
- **Japanese**: Tailscale接続の初期化に失敗しました
- **German**: Fehler beim Initialisieren der Tailscale-Verbindung
- **French**: Échec de l'initialisation de la connexion Tailscale

## Implementation Details

### Thread Safety
- **Concurrent access protection** with read/write mutexes
- **Race condition prevention** in multi-threaded environments
- **Atomic language switching** without data corruption

### Performance
- **Lazy initialization** - messages loaded only when needed
- **Memory efficient** - translations cached after first use
- **Fast lookup** - O(1) language detection and message retrieval

### Extensibility
- **Modular design** for easy addition of new languages
- **Standardized key system** for consistent translation management
- **Automated testing** for translation completeness

## Testing

### Automated Tests
```bash
# Test all language support
go test ./... -run TestI18n

# Test specific language normalization
go test ./... -run TestI18nLanguageNormalization

# Test new language translations
go test ./... -run TestI18nNewLanguages
```

### Manual Testing
```bash
# Test each language individually
for lang in en es zh hi ar bn pt ru ja de fr; do
  echo "Testing $lang:"
  ts-ssh --lang=$lang --help | head -3
  echo
done
```

## Contributing Translations

### Adding New Languages
1. **Add language constant** in `i18n.go`
2. **Add to supported languages map** with language.Tag
3. **Add normalization rules** in `normalizeLanguage()`
4. **Add core message translations** in `registerMessages()`
5. **Add test cases** for the new language
6. **Update documentation** in README and guides

### Translation Guidelines
- **Maintain technical accuracy** for SSH/networking terms
- **Use native script** when applicable (Arabic, Chinese, etc.)
- **Follow cultural conventions** for formal/informal address
- **Keep messages concise** to fit terminal displays
- **Test with native speakers** when possible

## Localization Best Practices

### For Users
- Set `TS_SSH_LANG` in your shell profile for consistent experience
- Use full language codes (`zh-CN`) for region-specific variants
- Test in non-ASCII languages to verify terminal Unicode support

### For Administrators
- Set language in environment for server deployments
- Use English (`en`) for automation and CI/CD consistency
- Document language settings in deployment guides

### For Developers
- Always use `T()` function for user-facing strings
- Test with multiple languages during development
- Consider text expansion in non-Latin scripts
- Validate Unicode handling in string operations

## Future Enhancements

### Planned Features
- **Automatic language detection** from user locale
- **Language-specific date/time formatting**
- **Localized error codes** and documentation links
- **Regional configuration defaults** (time zones, date formats)

### Community Contributions
- **Translation improvements** from native speakers
- **Additional language support** beyond top 11
- **Regional variants** (e.g., Brazilian vs European Portuguese)
- **Accessibility features** for different script directions