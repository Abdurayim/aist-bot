package config

import "testing"

// TestParseAdminPhones covers the formats a school administrator is likely
// to paste into .env: spaces, dashes, missing prefixes and duplicates.
func TestParseAdminPhones(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "plain list",
			raw:  "+998901112233,+998935556677",
			want: []string{"+998901112233", "+998935556677"},
		},
		{
			name: "four admins",
			raw:  "+998901112233,+998935556677,+998977778899,+998901234567",
			want: []string{"+998901112233", "+998935556677", "+998977778899", "+998901234567"},
		},
		{
			name: "spaces and dashes",
			raw:  " +998 90 111 22 33 , +998-93-555-66-77 ",
			want: []string{"+998901112233", "+998935556677"},
		},
		{
			name: "missing plus and national form",
			raw:  "998901112233,935556677",
			want: []string{"+998901112233", "+998935556677"},
		},
		{
			name: "duplicates are collapsed",
			raw:  "+998901112233,+998 90 111 22 33",
			want: []string{"+998901112233"},
		},
		{
			name: "semicolons also separate",
			raw:  "+998901112233;+998935556677",
			want: []string{"+998901112233", "+998935556677"},
		},
		{
			name: "empty",
			raw:  "  ",
			want: []string{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseAdminPhones(c.raw)

			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}

			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("got %v, want %v", got, c.want)
				}
			}
		})
	}
}

// TestValidateRequiresTheEssentials keeps the bot from starting in a state
// where appeals would go nowhere.
func TestValidateRequiresTheEssentials(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "no token",
			cfg:     Config{Admin: AdminConfig{PhoneNumbers: []string{"+998901112233"}}},
			wantErr: true,
		},
		{
			name:    "no admins",
			cfg:     Config{Bot: BotConfig{Token: "t"}},
			wantErr: true,
		},
		{
			name: "four admins is fine",
			cfg: Config{
				Bot: BotConfig{Token: "t"},
				Admin: AdminConfig{PhoneNumbers: []string{
					"+998901112233", "+998935556677", "+998977778899", "+998901234567",
				}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cfg.Validate()

			if c.wantErr && err == nil {
				t.Fatal("expected an error")
			}

			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsAdminPhone(t *testing.T) {
	admin := AdminConfig{PhoneNumbers: ParseAdminPhones("+998901112233,+998935556677")}

	if !admin.IsAdminPhone("+998935556677") {
		t.Fatal("a listed number must be recognised")
	}

	if admin.IsAdminPhone("+998900000000") {
		t.Fatal("an unlisted number must not be recognised")
	}
}
