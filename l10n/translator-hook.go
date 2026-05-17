package l10n

var hooks map[string]*TranslatorHook

type TranslatorHook struct {
	t      *Translator
	prefix string
}

func NewTranslatorHook(prefix string) *TranslatorHook {
	if _, ok := hooks[prefix]; !ok {
		hooks[prefix] = &TranslatorHook{T, prefix}
	}
	return hooks[prefix]
}

func (h *TranslatorHook) Set(key string, val any, toOverwrite bool) (alreadySet bool, err error) {
	return h.t.Set(h.prefix+key, val, toOverwrite)
}

func (h *TranslatorHook) SetPath(val any, toOverwrite bool, path ...any) (alreadySet bool, err error) {
	return h.t.SetPath(val, toOverwrite, append([]any{h.prefix}, path...)...)
}

func (h *TranslatorHook) Get(key string) any {
	return h.t.Get(h.prefix + "." + key)
}

func (h *TranslatorHook) GetPath(path ...any) any {
	return h.t.GetPath(append([]any{h.prefix}, path...)...)
}

func (h *TranslatorHook) GetDefault(key string, def any) any {
	return h.t.GetDefault(h.prefix+"."+key, def)
}

func (h *TranslatorHook) GetDefaultPath(def any, path ...any) any {
	return h.t.GetDefaultPath(def, append([]any{h.prefix}, path...)...)
}
