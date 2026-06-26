package cli

// Version holds the current binary version. It defaults to "unknown" for
// locally-built binaries and is overridden via ldflags at release time:
//
//	go build -ldflags "-X main.version=v1.2.3"
//
// main.go calls SetVersion() to push the value here.
var Version = "unknown"

// SetVersion sets the package-level Version variable. Called from main()
// with the ldflags-injected value.
func SetVersion(v string) {
	if v != "" {
		Version = v
	}
}
