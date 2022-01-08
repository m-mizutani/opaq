package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zlog"
	"github.com/m-mizutani/zlog/filter"
	"github.com/urfave/cli/v2"
)

var logger = zlog.New()

type Proc struct {
	httpClient HTTPClient
	stdin      io.Reader
	stdout     io.Writer
}

type Option func(proc *Proc)

func New(options ...Option) *Proc {
	proc := &Proc{
		httpClient: &http.Client{},
		stdin:      os.Stdin,
		stdout:     os.Stdout,
	}
	for _, opt := range options {
		opt(proc)
	}
	return proc
}

type config struct {
	queryConfig

	headers  cli.StringSlice
	LogLevel string
}

func (x *Proc) Cmd(ctx context.Context, args []string) error {
	var cfg config

	app := &cli.App{
		Name:  "opaq",
		Usage: "Query to OPA server",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "fail-defined",
				Usage:       "exits with non-zero exit code on undefined/empty result and errors",
				Destination: &cfg.FailDefined,
			},
			&cli.BoolFlag{
				Name:        "fail-undefined",
				Usage:       "exits with non-zero exit code on defined/non-empty result and errors",
				Destination: &cfg.FailUndefined,
			},

			&cli.StringFlag{
				Name:        "url",
				Aliases:     []string{"u"},
				EnvVars:     []string{"OPAQ_URL"},
				Required:    true,
				Usage:       "Query URL of OPA server, e.g. https://opa.example.com/v1/data/foo",
				Destination: &cfg.URL,
			},

			&cli.StringFlag{
				Name:        "input",
				Aliases:     []string{"i"},
				Usage:       "input file, `-` is stdin",
				Value:       "-",
				Destination: &cfg.Input,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "output file, `-` is stdout",
				Value:       "-",
				Destination: &cfg.Output,
			},

			&cli.StringSliceFlag{
				Name:        "http-header",
				Aliases:     []string{"H"},
				EnvVars:     []string{"OPAQ_HEADER"},
				Usage:       "Custom header(s) of a HTTP request",
				Destination: &cfg.headers,
			},

			&cli.StringFlag{
				Name:        "log-level",
				Aliases:     []string{"l"},
				Usage:       "logging level [debug,info,warn,error]",
				Value:       "info",
				Destination: &cfg.LogLevel,
			},
		},

		Before: func(_ *cli.Context) error {
			cfg.Headers = cfg.headers.Value()

			l, err := zlog.NewWithError(
				zlog.WithLogLevel(cfg.LogLevel),
				zlog.WithFilters(filter.Tag()),
			)
			if err != nil {
				return err
			}
			logger = l

			logger.With("config", cfg).Debug("starting")

			return nil
		},
		After: func(_ *cli.Context) error {
			logger.Debug("exiting")
			return nil
		},

		Action: func(_ *cli.Context) error {
			return x.query(ctx, &cfg.queryConfig)
		},
	}

	if err := app.Run(args); err != nil {
		if errors.Is(ErrExitWithNonZero, err) {
			return err
		}

		log := logger.Log()
		var goErr *goerr.Error
		if errors.As(err, &goErr) {
			for key, value := range goErr.Values() {
				log.With(key, value)
			}
		}

		log.Error(err.Error())
		logger.With("config", cfg).Err(err).Debug("error detail")
		return err
	}

	return nil
}
