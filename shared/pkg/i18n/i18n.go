package i18n

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localesFS embed.FS

var (
	defaultBundle *Bundle
	once          sync.Once
)

// Bundle 国际化消息包
type Bundle struct {
	mu       sync.RWMutex
	locale   string
	messages map[string]map[string]string // locale -> key -> message
	fallback string
}

// Init 初始化国际化系统
func Init(localesDir string, defaultLocale string) error {
	var err error
	once.Do(func() {
		defaultBundle = &Bundle{
			locale:   defaultLocale,
			messages: make(map[string]map[string]string),
			fallback: "en",
		}
		// Load embedded locales first
		err = defaultBundle.loadEmbeddedLocales()
	})
	return err
}

// loadEmbeddedLocales loads locales from embedded FS
func (b *Bundle) loadEmbeddedLocales() error {
	files, err := localesFS.ReadDir("locales")
	if err != nil {
		return fmt.Errorf("read embedded locales: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		locale := strings.TrimSuffix(file.Name(), ".yaml")
		path := "locales/" + file.Name()

		data, err := localesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded locale file %s: %w", path, err)
		}

		messages := make(map[string]string)
		if err := yaml.Unmarshal(data, &messages); err != nil {
			return fmt.Errorf("parse locale file %s: %w", path, err)
		}

		b.mu.Lock()
		b.messages[locale] = messages
		b.mu.Unlock()
	}

	return nil
}

// T 翻译消息（支持参数）
func T(key string, args ...interface{}) string {
	if defaultBundle == nil {
		return key
	}
	return defaultBundle.T(key, args...)
}

// T 翻译消息
func (b *Bundle) T(key string, args ...interface{}) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 尝试当前语言
	if msg, ok := b.messages[b.locale][key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}

	// 回退到英文
	if msg, ok := b.messages[b.fallback][key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}

	// 返回key本身
	return key
}

// SetLocale 设置当前语言
func SetLocale(locale string) {
	if defaultBundle != nil {
		defaultBundle.mu.Lock()
		defaultBundle.locale = locale
		defaultBundle.mu.Unlock()
	}
}

// GetLocale 获取当前语言
func GetLocale() string {
	if defaultBundle != nil {
		defaultBundle.mu.RLock()
		defer defaultBundle.mu.RUnlock()
		return defaultBundle.locale
	}
	return "en"
}
