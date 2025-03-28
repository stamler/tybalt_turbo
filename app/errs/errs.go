package errs

type HookError struct {
	Status  int                  `json:"status"` // Represents the HTTP status code
	Message string               `json:"message"`
	Data    map[string]CodeError `json:"data"`
}

func (e *HookError) Error() string {
	return e.Message
}

// CodeError is a custom error type that includes a code
type CodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func (e *CodeError) Error() string {
	return e.Message
}
