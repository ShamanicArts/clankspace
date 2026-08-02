package main

import (
	"context"
	"testing"
)

func TestTopLevelHelpDoesNotRequireConfiguration(t *testing.T) {
	if err := run(context.Background(), []string{"--help"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunHelpDoesNotRequireAClient(t *testing.T) {
	if err := runCommand(context.Background(), nil, []string{"--help"}); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), []string{"run", "--help"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsUnknownActionBeforeUsingClient(t *testing.T) {
	if err := runCommand(context.Background(), nil, []string{"status"}); err == nil {
		t.Fatal("expected an unsupported run action to fail")
	}
}
