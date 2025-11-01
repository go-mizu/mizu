//go:build !goexperiment.jsonv2

package mizu

import json "encoding/json"

type jsonEncoder = json.Encoder
type jsonDecoder = json.Decoder

var (
	newJSONDecoder = json.NewDecoder
	newJSONEncoder = json.NewEncoder
	jsonMarshal    = json.Marshal
)

func decDisallowUnknownFields(d *jsonDecoder)  { d.DisallowUnknownFields() }
func encSetEscapeHTML(e *jsonEncoder, on bool) { e.SetEscapeHTML(on) }
