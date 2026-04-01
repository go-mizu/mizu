//go:build go1.25 && goexperiment.jsonv2

package mizu

import json "encoding/json/v2"

type jsonEncoder = json.Encoder
type jsonDecoder = json.Decoder

var (
	newJSONDecoder = json.NewDecoder
	newJSONEncoder = json.NewEncoder
	jsonMarshal    = json.Marshal
)

func decDisallowUnknownFields(d *jsonDecoder)  { d.DisallowUnknownFields() }
func encSetEscapeHTML(e *jsonEncoder, on bool) { e.SetEscapeHTML(on) }
