package main

import "io"

// nolint
func WithHTTPClient(client HTTPClient) Option {
	return func(proc *Proc) {
		proc.httpClient = client
	}
}

// nolint
func WithStdin(stdin io.Reader) Option {
	return func(proc *Proc) {
		proc.stdin = stdin
	}
}
