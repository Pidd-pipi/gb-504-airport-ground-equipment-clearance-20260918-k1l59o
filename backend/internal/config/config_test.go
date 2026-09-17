package config

import "testing"

func TestProductionConfigurationRedLines(t *testing.T) {
	valid := &Config{AppEnv: "production", JWTSecret: "0123456789abcdef0123456789abcdef", CORSOrigins: []string{"https://ops.example.com"}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid production configuration rejected: %v", err)
	}

	cases := []*Config{
		{AppEnv: "production", JWTSecret: "short", CORSOrigins: []string{"https://ops.example.com"}},
		{AppEnv: "production", JWTSecret: "replace_with_at_least_32_random_characters", CORSOrigins: []string{"https://ops.example.com"}},
		{AppEnv: "production", JWTSecret: "0123456789abcdef0123456789abcdef", CORSOrigins: []string{"*"}},
		{AppEnv: "production", JWTSecret: "0123456789abcdef0123456789abcdef", CORSOrigins: []string{"https://ops.example.com"}, SeedDemoData: true},
	}
	for index, cfg := range cases {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("unsafe production configuration %d was accepted", index)
		}
	}
}
