package auth

type Validator struct {
	validKeys map[string]bool
}

func NewValidator(keys []string) *Validator {
	validKeys := make(map[string]bool)
	for _, key := range keys {
		validKeys[key] = true
	}
	return &Validator{validKeys: validKeys}
}

func (v *Validator) IsValid(key string) bool {
	return v.validKeys[key]
}
