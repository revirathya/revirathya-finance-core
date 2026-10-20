package config_test

import (
	"strings"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/config"
)

func TestLoadAppEnv(t *testing.T) {
	// unsetToggles clears the explicit settings, so only APP_ENV decides.
	unsetToggles := func(t *testing.T) {
		t.Helper()
		t.Setenv("GRPC_ENABLE_REFLECTION", "")
		t.Setenv("GQL_ENABLE_INTROSPECTION", "")
	}

	t.Run("unset is prod: reflection and introspection off", func(t *testing.T) {
		setRequired(t)
		unsetToggles(t)
		t.Setenv("APP_ENV", "")

		cfg, err := config.Load()

		if err != nil {
			t.Fatalf("unexpected error: %v (the default must be a valid APP_ENV)", err)
		}
		if cfg.AppEnv != "prod" || cfg.GrpcEnableReflection || cfg.GqlEnableIntrospection {
			t.Fatalf("got AppEnv %q, reflection %v, introspection %v; want prod with both off",
				cfg.AppEnv, cfg.GrpcEnableReflection, cfg.GqlEnableIntrospection)
		}
	})

	t.Run("dev turns reflection and introspection on by default", func(t *testing.T) {
		setRequired(t)
		unsetToggles(t)
		t.Setenv("APP_ENV", "dev")

		cfg, err := config.Load()

		if err != nil || cfg.AppEnv != "dev" || !cfg.GrpcEnableReflection || !cfg.GqlEnableIntrospection {
			t.Fatalf("got %+v, %v; want dev with both on", cfg, err)
		}
	})

	t.Run("explicit settings win over the APP_ENV defaults", func(t *testing.T) {
		for _, tt := range []struct{ env, value string }{{"dev", "false"}, {"prod", "true"}} {
			setRequired(t)
			t.Setenv("APP_ENV", tt.env)
			t.Setenv("GRPC_ENABLE_REFLECTION", tt.value)
			t.Setenv("GQL_ENABLE_INTROSPECTION", tt.value)

			cfg, err := config.Load()

			want := tt.value == "true"
			if err != nil || cfg.GrpcEnableReflection != want || cfg.GqlEnableIntrospection != want {
				t.Fatalf("APP_ENV=%s with both set to %s: got reflection %v, introspection %v, %v",
					tt.env, tt.value, cfg.GrpcEnableReflection, cfg.GqlEnableIntrospection, err)
			}
		}
	})

	t.Run("an unknown APP_ENV is a startup error naming the variable", func(t *testing.T) {
		for _, v := range []string{"production", "staging", "DEV"} {
			setRequired(t)
			t.Setenv("APP_ENV", v)

			_, err := config.Load()

			if err == nil || !strings.Contains(err.Error(), "APP_ENV") {
				t.Fatalf("APP_ENV=%q: got %v, want an error naming APP_ENV", v, err)
			}
		}
	})
}
