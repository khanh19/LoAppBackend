package users

import (
	"errors"
	"testing"

	"encore.dev/beta/errs"
)

func TestNormalizeUpsertUserProfileRequest(t *testing.T) {
	country := " +1 "
	national := "(415) 555-1234"
	dob := "2000-01-02"
	username := " My.User_01 "
	req := &UpsertUserProfileRequest{
		FirstName:           " Jane ",
		LastName:            " Doe ",
		Username:            username,
		PhoneCountryCode:    &country,
		PhoneNationalNumber: &national,
		DateOfBirth:         &dob,
	}

	normalized, err := normalizeUpsertUserProfileRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	if normalized.FirstName != "Jane" || normalized.LastName != "Doe" {
		t.Fatalf("unexpected names: %#v", normalized)
	}
	if normalized.Username != "my.user_01" {
		t.Fatalf("username = %q", normalized.Username)
	}
	if normalized.PhoneE164 == nil || *normalized.PhoneE164 != "+14155551234" {
		t.Fatalf("phone_e164 = %#v", normalized.PhoneE164)
	}
	if normalized.PhoneNationalNumber == nil || *normalized.PhoneNationalNumber != "4155551234" {
		t.Fatalf("phone_national_number = %#v", normalized.PhoneNationalNumber)
	}
	if normalized.DateOfBirth == nil || *normalized.DateOfBirth != "2000-01-02" {
		t.Fatalf("date_of_birth = %#v", normalized.DateOfBirth)
	}
}

func TestNormalizeUpsertUserProfileRequestRequiresPhonePair(t *testing.T) {
	country := "+1"
	req := &UpsertUserProfileRequest{
		FirstName:        "Jane",
		LastName:         "Doe",
		Username:         "jane_doe",
		PhoneCountryCode: &country,
	}

	_, err := normalizeUpsertUserProfileRequest(req)
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

func TestNormalizeUsernameValidation(t *testing.T) {
	if _, err := normalizeUsername("ab"); err == nil {
		t.Fatal("expected too-short username error")
	}
	if _, err := normalizeUsername("abc$def"); err == nil {
		t.Fatal("expected invalid chars error")
	}
}
