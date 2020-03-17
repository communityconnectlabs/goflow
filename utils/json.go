package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
)

// JSONMarshal marshals the given object to JSON
func JSONMarshal(v interface{}) ([]byte, error) {
	return jsonMarshal(v, "")
}

// JSONMarshalPretty marshals the given object to pretty JSON
func JSONMarshalPretty(v interface{}) ([]byte, error) {
	return jsonMarshal(v, "    ")
}

func jsonMarshal(v interface{}, indent string) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // see https://github.com/golang/go/issues/8592
	encoder.SetIndent("", indent)

	err := encoder.Encode(v)
	if err != nil {
		return nil, err
	}

	// don't include the final \n that .Encode() adds
	data := buffer.Bytes()
	return data[0 : len(data)-1], nil
}

// UnmarshalAndValidate is a convenience function to unmarshal an object and validate it
func UnmarshalAndValidate(data []byte, obj interface{}) error {
	err := json.Unmarshal(data, obj)
	if err != nil {
		return err
	}

	return Validate(obj)
}

// UnmarshalArray unmarshals an array of objects from the given JSON
func UnmarshalArray(data json.RawMessage) ([]json.RawMessage, error) {
	var items []json.RawMessage
	err := json.Unmarshal(data, &items)
	return items, err
}

// UnmarshalAndValidateWithLimit unmarsmals a struct with a limit on how many bytes can be read from the given reader
func UnmarshalAndValidateWithLimit(reader io.ReadCloser, s interface{}, limit int64) error {
	body, err := ioutil.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return err
	}
	if err := reader.Close(); err != nil {
		return err
	}
	if err := json.Unmarshal(body, &s); err != nil {
		return err
	}

	// validate the request
	return Validate(s)
}

// JSONDecodeGeneric decodes the given JSON as a generic map or slice
func JSONDecodeGeneric(data []byte) (interface{}, error) {
	var asGeneric interface{}
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	decoder.UseNumber()
	return asGeneric, decoder.Decode(&asGeneric)
}

// Typed is an interface of objects that are marshalled as typed envelopes
type Typed interface {
	Type() string
}

// TypedEnvelope can be mixed into envelopes that have a type field
type TypedEnvelope struct {
	Type string `json:"type" validate:"required"`
}

// ReadTypeFromJSON reads a field called `type` from the given JSON
func ReadTypeFromJSON(data []byte) (string, error) {
	t := &TypedEnvelope{}
	if err := UnmarshalAndValidate(data, t); err != nil {
		return "", err
	}
	return t.Type, nil
}

// ExtractResponseJSON extracts a JSON body from an HTTP response trace
func ExtractResponseJSON(response []byte) json.RawMessage {
	parts := bytes.SplitN(response, []byte("\r\n\r\n"), 2)

	// this response doesn't have a body
	if len(parts) != 2 || len(parts[1]) == 0 {
		return nil
	}

	body := parts[1]

	// check if body is valid JSON and can be returned as is
	if json.Valid(body) {
		return body
	}

	// if not, treat body as text and encode as a JSON string
	asString, _ := json.Marshal(string(body))
	return asString
}
