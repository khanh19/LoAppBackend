package onboarding

import (
	"errors"
	"testing"

	"encore.dev/beta/errs"
)

func TestValidateSaveStepRequestCategoriesMaxThree(t *testing.T) {
	err := validateSaveStepRequest(2, &SaveStepRequest{
		CategoryIDs: []string{"a", "b", "c", "d"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var encoreErr *errs.Error
	if !errors.As(err, &encoreErr) {
		t.Fatalf("error type = %T", err)
	}
	if encoreErr.Code != errs.InvalidArgument {
		t.Fatalf("error code = %v", encoreErr.Code)
	}
}

func TestValidateSaveStepRequestStepOneRequiresCity(t *testing.T) {
	err := validateSaveStepRequest(1, &SaveStepRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatPriceLevel(t *testing.T) {
	level := int16(3)
	if got := formatPriceLevel(&level); got != "$20–$30" {
		t.Fatalf("price = %q", got)
	}
}
