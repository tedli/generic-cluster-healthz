package utils

func Pointer[V any](v V) *V {
	return &v
}
