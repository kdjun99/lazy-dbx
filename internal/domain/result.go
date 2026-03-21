package domain

// Result carries either a value or an error from a service method.
// All service methods return Result[T] to make error handling explicit.
type Result[T any] struct {
	Data  T
	Error error
}
