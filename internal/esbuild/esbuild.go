package esbuild

import (
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

type (
	BuildOptions = api.BuildOptions
	BuildResult  = api.BuildResult
	OnLoadResult = api.OnLoadResult
	OnLoadArgs   = api.OnLoadArgs
	EntryPoint   = api.EntryPoint
	OutputFile   = api.OutputFile
	Plugin       = api.Plugin
	StdinOptions = api.StdinOptions
)

var (
	JSXAutomatic    = api.JSXAutomatic
	FormatIIFE      = api.FormatIIFE
	FormatESModule  = api.FormatESModule
	FormatCommonJS  = api.FormatCommonJS
	PlatformNeutral = api.PlatformNeutral
	PlatformBrowser = api.PlatformBrowser
	LoaderJS        = api.LoaderJS
	LoaderTSX       = api.LoaderTSX
	ES2015          = api.ES2015
	ES2023          = api.ES2023
)

func Build(options BuildOptions) (*BuildResult, error) {
	result := api.Build(options)
	if len(result.Errors) > 0 {
		return nil, &Error{result.Errors}
	}
	return &result, nil
}

type Error struct {
	messages []api.Message
}

func (e *Error) Error() string {
	errors := api.FormatMessages(e.messages, api.FormatMessagesOptions{
		Color:         true,
		Kind:          api.ErrorMessage,
		TerminalWidth: 120,
	})
	return strings.Join(errors, "\n")
}
