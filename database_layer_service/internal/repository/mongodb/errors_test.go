package mongodb

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"job-management-service/internal/repository"
)

// Erorile driverului sunt construite de mana, deci testul nu are nevoie de MongoDB pornit.
func TestConvertError(t *testing.T) {
	duplicate := mongo.WriteException{WriteErrors: []mongo.WriteError{{
		Code:    11000,
		Message: `E11000 duplicate key error collection: db.computers index: IP dup key: { IP: "10.0.0.1" }`,
	}}}

	tests := []struct {
		name      string
		err       error
		want      error  // eroarea din repository pe care o asteptam (errors.Is)
		wantField string // pentru DuplicateError
	}{
		{"document inexistent", mongo.ErrNoDocuments, repository.ErrNotFound, ""},
		{"cheie duplicata", duplicate, nil, "IP"},
		{"timeout", context.DeadlineExceeded, repository.ErrUnavailable, ""},
		{"retea cazuta", mongo.CommandError{Labels: []string{"NetworkError"}}, repository.ErrUnavailable, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertError(tt.err)
			var dup *repository.DuplicateError
			switch {
			case tt.wantField != "":
				if !errors.As(got, &dup) || dup.Field != tt.wantField {
					t.Errorf("convertError = %v, vrem DuplicateError pe %q", got, tt.wantField)
				}
			case !errors.Is(got, tt.want):
				t.Errorf("convertError = %v, vrem %v", got, tt.want)
			}
		})
	}

	if convertError(nil) != nil {
		t.Error("convertError(nil) trebuie sa fie nil")
	}
}
