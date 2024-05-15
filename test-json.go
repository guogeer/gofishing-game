package main

import "encoding/json"

func main() {
	m := map[string]any{
		"a": 1,
		"b": 1.1,
		"c": "ab",
	}

	b, _ := json.Marshal(m)

	m1 := map[string]json.RawMessage{}
	json.Unmarshal(b, &m1)
	b1, _ := json.Marshal(m1)
	print(string(b), string(b1))
}
