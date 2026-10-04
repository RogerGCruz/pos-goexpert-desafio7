package test

import (
	"os"
	"testing"

	steps "pos-goexpert-desafio7-test/steps"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

func TestMain(m *testing.M) {
	status := godog.TestSuite{
		Name:                 "e2e",
		TestSuiteInitializer: nil,
		ScenarioInitializer:  steps.InitializeTestSuite,
		Options: &godog.Options{
			Format:        "pretty",
			Paths:         []string{"./features"},
			Randomize:     0,
			StopOnFailure: true,
			Output:        colors.Colored(os.Stdout),
		},
	}.Run()

	if status != 0 {
		os.Exit(status)
	}

	os.Exit(m.Run())
}
