package l10n

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var T *Translator

func init() {
	var err error
	if T, err = NewTranslator("ru", "en"); err != nil {
		slog.Error("fail to load localization", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

type Translator struct {
	l  map[string]any
	mu sync.RWMutex
}

func NewTranslator(locales ...string) (*Translator, error) {
	t := &Translator{l: make(map[string]any)}
	errs := make([]error, 0)

	for _, locale := range locales {
		contentBytes, err := os.ReadFile(fmt.Sprintf("l10n/%s.yaml", locale))
		if err != nil {
			errs = append(errs, fmt.Errorf("opening one of files: l10n/%s.yaml: %w", locale, err))
			continue
		}

		var content map[string]any
		if err := yaml.Unmarshal(contentBytes, &content); err != nil {
			errs = append(errs, fmt.Errorf("parsing one of files: l10n/%s.yaml: %w", locale, err))
			continue
		}

		errs = append(errs, t.initMap("."+locale, content)...)
	}

	return t, errors.Join(errs...)
}

func (t *Translator) Set(key string, val any, toOverwrite bool) (alreadySet bool, err error) {
	t.mu.RLock()
	if _, alreadySet = t.l[key]; !alreadySet || (alreadySet && toOverwrite) {
		t.mu.RUnlock()
		return alreadySet, errors.Join(t.setSafe(key, val)...)
	}
	t.mu.RUnlock()
	return alreadySet, nil
}

func (t *Translator) SetPath(val any, toOverwrite bool, path ...any) (alreadySet bool, err error) {
	key, err := formatPath(path...)
	if err != nil {
		return false, err
	}
	return t.Set(key, val, toOverwrite)
}

func (t *Translator) Get(key string) any {
	return t.GetDefault(key, key[strings.LastIndex(key, ".")+1:])
}

func (t *Translator) GetPath(path ...any) any {
	return t.GetDefaultPath(path[len(path)-1], path...)
}

func (t *Translator) GetDefault(key string, def any) any {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if v, ok := t.l[key]; ok {
		return v
	}
	slog.Warn("failed to find localization for a key", slog.String("key", key))
	return def
}

func (t *Translator) GetDefaultPath(def any, path ...any) any {
	t.mu.RLock()
	defer t.mu.RUnlock()
	key, err := formatPath(path...)
	if err != nil {
		slog.Error("get localized value by path: path is invalid", slog.String("error", err.Error()))
		return def
	}
	return t.GetDefault(key, def)
}

func (t *Translator) initSlice(parent string, level []any) []error {
	errs := make([]error, 0)
	for i, v := range level {
		newParent := parent + fmt.Sprintf("[%d]", i)
		t.setSafe(newParent, v)
	}
	return errs
}

func (t *Translator) initMap(parent string, level map[string]any) []error {
	errs := make([]error, 0)
	for k, v := range level {
		key, err := formatPath(k)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		errs = append(errs, t.setSafe(parent+key, v)...)
	}
	return errs
}

func (t *Translator) setSafe(k string, v any) []error {
	errs := make([]error, 0)
	t.mu.Lock()
	switch vv := v.(type) {
	case int:
		t.l[k] = vv
		t.mu.Unlock()
	case float64:
		t.l[k] = vv
		t.mu.Unlock()
	case string:
		t.l[k] = vv
		t.mu.Unlock()
	case []any:
		t.mu.Unlock()
		errs = append(errs, t.initSlice(k, vv)...)
	case map[string]any:
		t.mu.Unlock()
		errs = append(errs, t.initMap(k, vv)...)
	default:
		t.mu.Unlock()
		errs = append(errs, fmt.Errorf("invalid value type: %s, %v", k, vv))
	}
	return errs
}

func formatPath(path ...any) (string, error) {
	var key string
	for _, part := range path {
		switch k := part.(type) {
		case int:
			key += fmt.Sprintf("[%d]", k)
		case string:
			key += "." + strings.ReplaceAll(k, ".", "\\.")
		case []any:
			add, err := formatPath(k...)
			if err != nil {
				return "", fmt.Errorf("while formatting nested slice: %w", err)
			}
			key += add
		default:
			return "", fmt.Errorf("invalid key part: after %s follows %v", key, k)
		}
	}
	return key, nil
}
