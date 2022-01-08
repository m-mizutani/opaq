package main

import "github.com/m-mizutani/goerr"

var (
	ErrInvalidConfiguration = goerr.New("invalid configuration")
	ErrInvalidInput         = goerr.New("invalid input")
	ErrRequestFailed        = goerr.New("request to OPA server failed")
	ErrUnexpectedResp       = goerr.New("unexpected response from OPA server")

	// just to control exit code
	ErrExitWithNonZero = goerr.New("exit with non-zero")
)
