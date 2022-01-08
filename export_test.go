package main

import "io"

func WithHTTPClient(client HTTPClient) Option {
	return func(proc *Proc) {
		proc.httpClient = client
	}
}

func WithStdin(stdin io.Reader) Option {
	return func(proc *Proc) {
		proc.stdin = stdin
	}
}
