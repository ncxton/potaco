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

// ExitCoder is retained for backward compatibility with tests that check
// exit codes. New code should use UserError and the userErr/configUserErr/
// apiUserErr/imageUserErr constructors instead.
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
// This is a legacy constructor that creates a UserError with the raw error
// as the message. Prefer configUserErr for new code.
func configError(err error) error {
	return &UserError{Category: CatConfig, Message: err.Error(), Raw: err}
}

// apiError wraps an error so Execute() exits with the API exit code (3).
func apiError(err error) error {
	return &UserError{Category: CatAPI, Message: err.Error(), Raw: err}
}

// imageError wraps an error so Execute() exits with the image exit code (4).
func imageError(err error) error {
	return &UserError{Category: CatImage, Message: err.Error(), Raw: err}
}
