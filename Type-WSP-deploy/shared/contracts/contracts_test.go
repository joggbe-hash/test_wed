package contracts

import (
	"encoding/json"
	"testing"
)

func TestLoginOwnershipEmailContract(t *testing.T) {
	payload := LoginOwnershipEmailPayload{
		Purpose:            EmailPurposeLoginOwnership,
		Email:              "user@example.test",
		Code:               "ABCDEFGHJKLMNPQR",
		ChallengeID:        "challenge-123",
		DeliveryID:         "delivery-456",
		ExpiresAtUnixMilli: 1_800_000_000_000,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	want := `{"purpose":"login_ownership","email":"user@example.test","code":"ABCDEFGHJKLMNPQR","challenge_id":"challenge-123","delivery_id":"delivery-456","expires_at_unix_milli":1800000000000}`
	if string(encoded) != want {
		t.Fatalf("encoded payload = %s; want %s", encoded, want)
	}

	var decoded LoginOwnershipEmailPayload
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded != payload {
		t.Fatalf("decoded payload = %#v; want %#v", decoded, payload)
	}
}
