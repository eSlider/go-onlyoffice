package model_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestProjectStatusFromInt(t *testing.T) {
	if got := model.ProjectStatusFromAny(2); got != model.ProjectLifecycleClosed {
		t.Fatalf("got %q", got)
	}
	if got := model.ProjectStatusFromAny(0); got != model.ProjectLifecycleOpen {
		t.Fatalf("got %q", got)
	}
}

func TestProjectStatusToggle(t *testing.T) {
	s := model.ProjectLifecycleOpen
	if s.Next() != model.ProjectLifecycleClosed {
		t.Fatal("expected closed")
	}
	if s.Next().Prev() != model.ProjectLifecycleOpen {
		t.Fatal("expected open after toggle back")
	}
}
