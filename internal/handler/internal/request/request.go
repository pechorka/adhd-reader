package request

import (
	"encoding/json"
	"io"
)

func DecodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
