package argo

// Err is the common base type for all errors that are reported by Argo package
// This can be used to quickly identify whether a returned error comes from Argo
type Err struct {
}

func (e Err) Error() string {
	return ""
}
