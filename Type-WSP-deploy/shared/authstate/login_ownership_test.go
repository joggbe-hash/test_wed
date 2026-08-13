package authstate

import "testing"

func TestLoginOwnershipKeyNormalizationVectors(t *testing.T) {
	const digest = "5d948c5573e1d347ff925914610d481638347f79ef65a44a1ec29b05a24451c3"
	vectors := []struct {
		name   string
		build  func(string) string
		prefix string
	}{
		{"active", LoginOwnershipActiveKey, LoginOwnershipActiveKeyPrefix},
		{"cooldown", LoginOwnershipSendCooldownKey, LoginOwnershipSendCooldownKeyPrefix},
		{"hourly", LoginOwnershipSendHourlyKey, LoginOwnershipSendHourlyKeyPrefix},
		{"daily", LoginOwnershipSendDailyKey, LoginOwnershipSendDailyKeyPrefix},
	}
	for _, vector := range vectors {
		t.Run(vector.name, func(t *testing.T) {
			want := vector.prefix + digest
			for _, input := range []string{"user@example.test", " User@Example.Test ", "USER@EXAMPLE.TEST"} {
				if got := vector.build(input); got != want {
					t.Fatalf("key for %q = %q; want %q", input, got, want)
				}
			}
		})
	}
}

func TestLoginOwnershipActiveHashFieldNames(t *testing.T) {
	fields := []struct {
		got  string
		want string
	}{
		{LoginOwnershipChallengeIDField, "challenge_id"},
		{LoginOwnershipCodeField, "code"},
		{LoginOwnershipDeliveryIDField, "delivery_id"},
		{LoginOwnershipDeliveryStateField, "delivery_state"},
	}
	for _, field := range fields {
		if field.got != field.want {
			t.Fatalf("field name = %q; want %q", field.got, field.want)
		}
	}
}
