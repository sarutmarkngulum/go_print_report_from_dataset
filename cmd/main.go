package main

import (
	"context"
	"log"
	"os"

	"go_print_report_from_dataset/internal/adapters/parser"
	"go_print_report_from_dataset/internal/adapters/renderer"
	"go_print_report_from_dataset/internal/application"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic: %v", err)
			os.Exit(1)
		}
	}()

	if err := run(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	config := application.DefaultConfig()
	datasetParser := parser.NewJSONParser()
	previewRenderer, err := renderer.NewHTMLRenderer()
	if err != nil {
		return err
	}

	app := application.New(config, datasetParser, previewRenderer, nil)
	if err := app.Run(context.Background()); err != nil {
		return err
	}
	return nil
}
