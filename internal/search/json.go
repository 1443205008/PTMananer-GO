package search

import "encoding/json"

func jsonUnmarshalImpl(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
