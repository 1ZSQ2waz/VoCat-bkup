package store

import (
	"context"
	"errors"
	"testing"
)

func TestPhoneAssociationPersistsByICCID(t *testing.T) {
	database, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	value := PhoneAssociation{
		ICCID:    "89441000400311061404",
		DeviceID: "ec20",
		Number:   "+447700900123",
		Source:   "ims_p_associated_uri",
	}
	if err := database.UpsertPhoneAssociation(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	got, err := database.PhoneAssociation(context.Background(), value.ICCID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != value.Number || got.Source != value.Source || got.DeviceID != value.DeviceID {
		t.Fatalf("PhoneAssociation() = %#v", got)
	}

	value.Number = "+447700900456"
	if err := database.UpsertPhoneAssociation(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	got, err = database.PhoneAssociation(context.Background(), value.ICCID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != value.Number {
		t.Fatalf("updated number = %q", got.Number)
	}
	if _, err := database.PhoneAssociation(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}
