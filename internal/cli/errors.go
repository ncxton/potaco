package cli

import "fmt"

// Exit codes as defined by the potaco v0 spec:
//
//	0 = success
//	2 = config error
//	3 = API error
//	4 = image error
const (
	ExitSuccess = 0
	ExitConfig  = 2
	ExitAPI     = 3
	ExitImage   = 4
)

// ExitCoder is an error that carries a specific process exit code.
// It wraps an inner error so that callers can inspect the underlying
// cause while still signaling the desired exit status to Execute().
type ExitCoder struct {
	Code int
	Err  error
}

func (e *ExitCoder) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitCoder) Unwrap() error {
	return e.Err
}

// configError wraps an error so Execute() exits with the config exit code (2).
func configError(err error) error {
	return &ExitCoder{Code: ExitConfig, Err: err}
}

// apiError wraps an error so Execute() exits with the API exit code (3).
func apiError(err error) error {
	return &ExitCoder{Code: ExitAPI, Err: err}
}

// imageError wraps an error so Execute() exits with the image exit code (4).
func imageError(err error) error {
	return &ExitCoder{Code: ExitImage, Err: err}
}
