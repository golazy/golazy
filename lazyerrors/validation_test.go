package lazyerrors_test

import (
	"errors"
	"strings"
	"testing"

	"golazy.dev/lazyerrors"
)

type validationPostForm struct {
	Title   string `validate:"presence;min:3;max:10"`
	Body    string `schema:"body_text" validate:"presence"`
	Ignored string `schema:"-" validate:"presence"`
}

func TestValidatorUsesValidateTagsAndSchemaNames(t *testing.T) {
	err := lazyerrors.Validator(validationPostForm{
		Title: "Hi",
	})
	if err == nil {
		t.Fatal("Validator error = nil, want validation errors")
	}

	validations := lazyerrors.ValidationErrors(err)
	if got, want := len(validations), 2; got != want {
		t.Fatalf("ValidationErrors length = %d, want %d: %#v", got, want, validations)
	}

	titleErrors := lazyerrors.FieldErrorsFor(err, "title")
	if got, want := len(titleErrors), 1; got != want {
		t.Fatalf("title errors = %d, want %d", got, want)
	}
	if titleErrors[0].Type != lazyerrors.ValidationMin {
		t.Fatalf("title error type = %q, want %q", titleErrors[0].Type, lazyerrors.ValidationMin)
	}
	var minErr lazyerrors.MinSizeErr
	if !errors.As(titleErrors[0], &minErr) {
		t.Fatalf("title error does not wrap MinSizeErr: %#v", titleErrors[0])
	}
	if minErr.Min != 3 || minErr.Current != 2 {
		t.Fatalf("MinSizeErr = %#v, want Min 3 Current 2", minErr)
	}

	bodyErrors := lazyerrors.FieldErrorsFor(err, "body_text")
	if got, want := len(bodyErrors), 1; got != want {
		t.Fatalf("body_text errors = %d, want %d", got, want)
	}
	if bodyErrors[0].Type != lazyerrors.ValidationPresence {
		t.Fatalf("body_text error type = %q, want %q", bodyErrors[0].Type, lazyerrors.ValidationPresence)
	}
	var presenceErr lazyerrors.PresenceErr
	if !errors.As(bodyErrors[0], &presenceErr) {
		t.Fatalf("body_text error does not wrap PresenceErr: %#v", bodyErrors[0])
	}

	if ignored := lazyerrors.FieldErrorsFor(err, "ignored"); len(ignored) != 0 {
		t.Fatalf("ignored errors = %#v, want none", ignored)
	}
}

type validationCustomForm struct {
	Name string `validate:"presence"`
}

func (validationCustomForm) Validate() error {
	return lazyerrors.NewValidationError("confirm", "match", errors.New("does not match"))
}

func TestValidatorJoinsCustomValidateError(t *testing.T) {
	err := lazyerrors.Validator(validationCustomForm{})
	if err == nil {
		t.Fatal("Validator error = nil, want validation errors")
	}

	grouped := lazyerrors.ErrorsFor(err)
	if got, want := len(grouped["name"]), 1; got != want {
		t.Fatalf("name errors = %d, want %d", got, want)
	}
	if got, want := len(grouped["confirm"]), 1; got != want {
		t.Fatalf("confirm errors = %d, want %d", got, want)
	}
	if grouped["confirm"][0].Type != "match" {
		t.Fatalf("confirm type = %q, want match", grouped["confirm"][0].Type)
	}
}

type validationSizedForm struct {
	Title string `validate:"max:3"`
}

func TestValidatorReportsMaxSize(t *testing.T) {
	err := lazyerrors.Validator(&validationSizedForm{Title: "hello"})
	if err == nil {
		t.Fatal("Validator error = nil, want max validation error")
	}
	fieldErrors := lazyerrors.FieldErrorsFor(err, "title")
	if got, want := len(fieldErrors), 1; got != want {
		t.Fatalf("title errors = %d, want %d", got, want)
	}
	var maxErr lazyerrors.MaxSizeErr
	if !errors.As(fieldErrors[0], &maxErr) {
		t.Fatalf("title error does not wrap MaxSizeErr: %#v", fieldErrors[0])
	}
	if maxErr.Max != 3 || maxErr.Current != 5 {
		t.Fatalf("MaxSizeErr = %#v, want Max 3 Current 5", maxErr)
	}
}

type validationBadTagForm struct {
	Title string `validate:"min:nope"`
}

func TestValidatorReturnsNormalErrorsForInvalidTags(t *testing.T) {
	err := lazyerrors.Validator(validationBadTagForm{})
	if err == nil {
		t.Fatal("Validator error = nil, want invalid tag error")
	}
	if validations := lazyerrors.ValidationErrors(err); len(validations) != 0 {
		t.Fatalf("ValidationErrors = %#v, want none", validations)
	}
	if !strings.Contains(err.Error(), `invalid min validation limit "nope"`) {
		t.Fatalf("error = %q, want invalid limit message", err.Error())
	}
}

func TestHelpersIgnoreNilAndNonErrors(t *testing.T) {
	helpers := lazyerrors.Helpers()
	errorsFor := helpers["errors_for"].(func(any) map[string][]lazyerrors.ValidationError)
	fieldErrorsFor := helpers["field_errors_for"].(func(any, string) []lazyerrors.ValidationError)

	if got := errorsFor(nil); len(got) != 0 {
		t.Fatalf("errors_for nil = %#v, want none", got)
	}
	if got := fieldErrorsFor("not an error", "title"); len(got) != 0 {
		t.Fatalf("field_errors_for non-error = %#v, want none", got)
	}
}
