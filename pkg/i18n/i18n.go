package i18n

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"errors"

	"github.com/valyala/fasttemplate"
)

var ErrNotFound = errors.New("not found")

type translation struct {
	template *fasttemplate.Template
	text     string
}

func (t *translation) UnmarshalJSON(data []byte) error {
	var text string
	err := json.Unmarshal(data, &text)
	if err != nil {
		return err
	}
	t.text = text
	t.template, err = fasttemplate.NewTemplate(text, "{{", "}}")
	return err
}

type Localies struct {
	mu  *sync.RWMutex
	cms map[string]map[string]*translation // map[language_code]map[message_id]message
}

func New() *Localies {
	return &Localies{
		mu: &sync.RWMutex{},
	}
}

func (l *Localies) Load(path string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()

	l.mu.Lock()
	defer l.mu.Unlock()

	var translations map[string]map[string]*translation
	err = json.NewDecoder(f).Decode(&translations)
	if err != nil {
		return err
	}
	l.cms = translations
	return nil
}

func (l *Localies) Get(lang, id string) (string, error) {
	translation, ok := l.get(lang, id)
	if !ok {
		return "", ErrNotFound
	}
	return translation.text, nil
}

func (l *Localies) GetWithArgs(lang, id string, args map[string]string) (string, error) {
	translation, ok := l.get(lang, id)
	if !ok {
		return "", ErrNotFound
	}
	return translation.template.ExecuteFuncStringWithErr(func(w io.Writer, tag string) (int, error) {
		value, ok := args[tag]
		if !ok {
			return 0, fmt.Errorf("missing argument %s", tag)
		}
		return w.Write([]byte(value))
	})
}

func (l *Localies) get(lang, id string) (*translation, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	langMap, ok := l.cms[lang]
	if !ok {
		return nil, false
	}
	translation, ok := langMap[id]
	return translation, ok
}
