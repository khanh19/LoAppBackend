package onboarding

import (
	"errors"
	"testing"

	"encore.dev/beta/errs"
)

func TestValidateCompleteOnboardingRequestCategoriesMaxThree(t *testing.T) {
	err := validateCompleteOnboardingRequest(&CompleteOnboardingRequest{
		CityIDs:     []string{"city"},
		CategoryIDs: []string{"a", "b", "c", "d"},
		VibeIDs:     []string{"vibe"},
		PlaceIDs:    []string{"place"},
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

func TestValidateCompleteOnboardingRequestRequiresSelections(t *testing.T) {
	err := validateCompleteOnboardingRequest(&CompleteOnboardingRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateCompleteOnboardingRequestAllowsSkip(t *testing.T) {
	if err := validateCompleteOnboardingRequest(&CompleteOnboardingRequest{Skip: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatPriceLevel(t *testing.T) {
	level := int16(3)
	if got := formatPriceLevel(&level); got != "$20–$30" {
		t.Fatalf("price = %q", got)
	}
}
