package assets

// Transformer processes raw asset bytes for a given media type.
//
// mediaType is "text/css" or "application/javascript". Implementations should
// return src unchanged for unknown types, or return an error if strict.
type Transformer interface {
	Transform(src []byte, mediaType string) ([]byte, error)
}

// NoopTransformer returns input unchanged. It is the default when no
// transformer is configured.
type NoopTransformer struct{}

// Transform returns src unchanged.
func (NoopTransformer) Transform(src []byte, _ string) ([]byte, error) {
	return src, nil
}
