package watcher

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAndWatch(t *testing.T) {
	testFile := "/tmp/testfile" + strconv.Itoa(int(time.Now().UnixNano())) + ".json"
	t.Cleanup(func() {
		err := os.Remove(testFile)
		assert.NoError(t, err)
	})
	updateFile := func(data string) {
		err := os.WriteFile(testFile, []byte(data), os.ModePerm)
		require.NoError(t, err)
	}
	const initialData = `{"en": {"hello": "hello"}}`
	updateFile(initialData)

	loader := &mockLoader{}
	watcher, err := LoadAndWatch(testFile, loader)
	require.NoError(t, err)
	require.Equal(t, initialData, loader.lastLoaded)

	const updatedData = `{"en": {"hello": "hello"}, "cn": {"hello": "你好"}}`
	updateFile(updatedData)
	time.Sleep(100 * time.Millisecond) // wait for watcher to reload cms
	require.Equal(t, updatedData, loader.lastLoaded)

	err = watcher.Close()
	require.NoError(t, err)
	const finalData = `{"en": {"hello": "hello"}, "cn": {"hello": "你好"}, "de": {"hello": "hallo"}}`
	updateFile(finalData)
	time.Sleep(100 * time.Millisecond)               // wait for watcher to reload cms
	require.Equal(t, updatedData, loader.lastLoaded) // watcher is closed, so no reload
}

type mockLoader struct {
	lastLoaded string
}

func (m *mockLoader) Load(path string) error {
	date, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	m.lastLoaded = string(date)
	return nil
}
