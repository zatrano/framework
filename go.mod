module github.com/zatrano/framework/v3

go 1.25.0

require (
	github.com/zatrano/canvas v0.2.0
	github.com/zatrano/rawhttp v0.2.4
)

// Use v3.1.0 or later. v3.0.0 was published with local replace directives, so
// go install github.com/zatrano/framework/v3@v3.0.0 is rejected.
retract v3.0.0
