package security

import (
	"strings"
	"testing"
)

// A password that satisfies every ValidatePassword rule.
const strongPassword = "Str0ng!Passw0rd"

func TestHashPasswordAndComparePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "typical strong password", password: strongPassword},
		{name: "with spaces", password: "My Secret 1s Saf3!"},
		{name: "unicode letters", password: "Pässwörd1!"},
		{name: "long but within bcrypt limit", password: "Aa1!" + strings.Repeat("x", 60)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("HashPassword(%q) unexpected error: %v", tt.password, err)
			}
			if hash == "" {
				t.Fatal("HashPassword returned an empty hash")
			}
			if hash == tt.password {
				t.Fatal("HashPassword returned the plaintext password")
			}

			// NOTE arg order: ComparePassword(hashedPassword, plainPassword).
			if !ComparePassword(hash, tt.password) {
				t.Errorf("ComparePassword(hash, %q) = false, want true", tt.password)
			}

			// Hashing the same password twice must produce different hashes
			// (bcrypt salts), yet both must verify.
			hash2, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("second HashPassword: %v", err)
			}
			if hash2 == hash {
				t.Error("HashPassword produced identical hashes; salt is not random")
			}
			if !ComparePassword(hash2, tt.password) {
				t.Error("ComparePassword failed against the second hash")
			}
		})
	}
}

func TestHashPasswordRejectsInvalidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "too short", password: "Ab1!"},
		{name: "common password", password: "password"},
		{name: "missing special character", password: "Password1"},
		{name: "empty", password: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if err == nil {
				t.Fatalf("HashPassword(%q) = %q, want an error", tt.password, hash)
			}
			if hash != "" {
				t.Errorf("HashPassword(%q) returned hash %q alongside an error, want empty", tt.password, hash)
			}
		})
	}
}

func TestComparePasswordFailures(t *testing.T) {
	validHash, err := HashPassword(strongPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	tests := []struct {
		name     string
		hash     string
		password string
	}{
		{name: "wrong password", hash: validHash, password: "Wr0ng!Passw0rd"},
		{name: "password differing by case", hash: validHash, password: strings.ToLower(strongPassword)},
		{name: "empty password against valid hash", hash: validHash, password: ""},
		{name: "garbage hash", hash: "this-is-not-a-bcrypt-hash", password: strongPassword},
		{name: "empty hash", hash: "", password: strongPassword},
		{name: "truncated bcrypt hash", hash: validHash[:len(validHash)-5], password: strongPassword},
		{name: "plaintext stored as hash", hash: strongPassword, password: strongPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ComparePassword(tt.hash, tt.password) {
				t.Errorf("ComparePassword(%q, %q) = true, want false", tt.hash, tt.password)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errFrag  string
	}{
		{name: "valid strong password", password: strongPassword},
		{name: "valid at minimum length", password: "Aa1!bcde"},
		{name: "valid at maximum length", password: "Aa1!" + strings.Repeat("z", MaxPasswordLength-4)},
		{
			name:     "too short",
			password: "Aa1!bcd",
			wantErr:  true,
			errFrag:  "at least 8 characters",
		},
		{
			name:     "empty",
			password: "",
			wantErr:  true,
			errFrag:  "at least 8 characters",
		},
		{
			name:     "too long",
			password: "Aa1!" + strings.Repeat("z", MaxPasswordLength),
			wantErr:  true,
			errFrag:  "not exceed 72",
		},
		{
			name:     "missing uppercase",
			password: "lower1!pass",
			wantErr:  true,
			errFrag:  "uppercase",
		},
		{
			name:     "missing lowercase",
			password: "UPPER1!PASS",
			wantErr:  true,
			errFrag:  "lowercase",
		},
		{
			name:     "missing number",
			password: "NoDigits!Here",
			wantErr:  true,
			errFrag:  "number",
		},
		{
			name:     "missing special character",
			password: "NoSpecial123",
			wantErr:  true,
			errFrag:  "special character",
		},
		{
			name:     "common password: password",
			password: "password",
			wantErr:  true,
			errFrag:  "too common",
		},
		{
			name:     "common password: password123",
			password: "password123",
			wantErr:  true,
			errFrag:  "too common",
		},
		{
			name:     "common password: 12345678",
			password: "12345678",
			wantErr:  true,
			errFrag:  "too common",
		},
		{
			name:     "common password: iloveyou",
			password: "iloveyou",
			wantErr:  true,
			errFrag:  "too common",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errFrag) {
				t.Errorf("ValidatePassword(%q) error = %q, want it to mention %q", tt.password, err, tt.errFrag)
			}
		})
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name         string
		password     string
		wantStrength string
		wantScore    int
	}{
		{
			// len<8: no length points; only lowercase.
			name:         "very weak",
			password:     "abc",
			wantStrength: "weak",
			wantScore:    1,
		},
		{
			// len>=8: 1 length point + lowercase.
			name:         "lowercase only, 8 chars",
			password:     "abcdefgh",
			wantStrength: "weak",
			wantScore:    2,
		},
		{
			// len>=8: 1 + lower + upper + digit.
			name:         "fair",
			password:     "Abcdef12",
			wantStrength: "fair",
			wantScore:    4,
		},
		{
			// len>=8,>=12: 2 + lower + upper + digit + special.
			name:         "good",
			password:     "Abcdef12!xyz",
			wantStrength: "good",
			wantScore:    6,
		},
		{
			// len>=8,>=12,>=16: 3 + lower + upper + digit + special.
			name:         "strong",
			password:     "Abcdef12!xyzQRST",
			wantStrength: "strong",
			wantScore:    7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strength, score := ValidatePasswordStrength(tt.password)
			if score != tt.wantScore {
				t.Errorf("ValidatePasswordStrength(%q) score = %d, want %d", tt.password, score, tt.wantScore)
			}
			if strength != tt.wantStrength {
				t.Errorf("ValidatePasswordStrength(%q) strength = %q, want %q", tt.password, strength, tt.wantStrength)
			}
			if score < 0 || score > 7 {
				t.Errorf("ValidatePasswordStrength(%q) score = %d, out of range [0,7]", tt.password, score)
			}
		})
	}
}

func TestValidatePasswordStrengthIsMonotonic(t *testing.T) {
	// A strictly richer password must never score lower than a poorer one.
	ladder := []string{"abc", "abcdefgh", "Abcdef12", "Abcdef12!xyz", "Abcdef12!xyzQRST"}

	prev := -1
	for _, password := range ladder {
		_, score := ValidatePasswordStrength(password)
		if score < prev {
			t.Errorf("score for %q = %d, lower than the previous rung's %d", password, score, prev)
		}
		prev = score
	}
}
