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
)

var (
	JSXAutomatic    = api.JSXAutomatic
	FormatIIFE      = api.FormatIIFE
	FormatESModule  = api.FormatESModule
	PlatformNeutral = api.PlatformNeutral
	PlatformBrowser = api.PlatformBrowser
	LoaderTSX       = api.LoaderTSX
	ES2017          = api.ES2017
	ES2015          = api.ES2015
)

// func BuildAll(options BuildOptions) ([]OutputFile, *esmeta.File, error) {
// 	result := api.Build(options)
// 	if len(result.Errors) > 0 {
// 		return nil, nil, &Error{result.Errors}
// 	}
// 	metafile, err := esmeta.Parse(result.Metafile)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	return result.OutputFiles, metafile, nil
// }

// func BuildOne(options BuildOptions) (*OutputFile, error) {
// 	result := api.Build(options)
// 	if len(result.Errors) > 0 {
// 		return nil, &Error{result.Errors}
// 	} else if len(result.OutputFiles) == 0 {
// 		return nil, errors.New("esbuild: no output files generated")
// 	}
// 	return &result.OutputFiles[0], nil
// }

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
		Color: false,
		Kind:  api.ErrorMessage,
	})
	return strings.Join(errors, "\n")
}
