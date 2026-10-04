package app

import (
	"context"
	"os"
	"testing"
)

func TestLookupTemperatureWithoutKey(t *testing.T) {
	// Ensure WEATHERAPI_KEY is not set
	_ = os.Unsetenv("WEATHERAPI_KEY")
	_, err := lookupTemperatureByCity(context.Background(), "Sao Paulo")
	if err == nil {
		t.Fatalf("expected error when WEATHERAPI_KEY is not set")
	}
}
