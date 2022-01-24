package context

import (
	"os"
	"strings"
	"testing"
)

var (
	baseFuncCtx = `{
  "name": "function-test",
  "version": "v1.0.0"
}`
	funcCtxWithWrongRuntime = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "wrongRuntime"
}`
	funcCtxWithKnativeRuntime = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Knative"
}`
	funcCtxWithAsyncRuntime = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Async"
}`
	funcCtxWithCustomPort = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Async",
  "port": "12345"
}`
	funcCtxWithWrongPort = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Async",
  "port": "wrongPort"
}`
	funcCtxWithPlugins = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Async",
  "port": "12345",
  "prePlugins": ["plgA", "plgB", "plgC"],
  "postPlugins": ["plgC", "plgA"]
}`
	funcCtxWithTracingCfg = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Async",
  "port": "12345",
  "prePlugins": ["plgA", "plgB", "plgC"],
  "postPlugins": ["plgC", "plgA", "plgB"],
  "pluginsTracing": {
    "enable": true,
    "provider": {
      "name": "skywalking",
      "oapServer": "localhost:xxx"
    },
    "tags": {
      "func": "function-test",
      "layer": "faas",
      "tag1": "value1",
      "tag2": "value2"
    },
    "baggage": {
      "key": "sw8-correlation",
      "value": "base64(string key):base64(string value),base64(string key2):base64(string value2)"
    }
  }
}`
	funcCtxWithWrongTracingCfg = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Async",
  "port": "12345",
  "prePlugins": ["plgA", "plgB", "plgC"],
  "postPlugins": ["plgC", "plgA", "plgB"],
  "pluginsTracing": {
    "enable": true,
    "provider": {
      "name": "",
      "oapServer": "localhost:xxx"
    }
  }
}`
	funcCtxWithWrongTracingCfgProvider = `{
  "name": "function-test",
  "version": "v1.0.0",
  "runtime": "Async",
  "port": "12345",
  "prePlugins": ["plgA", "plgB", "plgC"],
  "postPlugins": ["plgC", "plgA", "plgB"],
  "pluginsTracing": {
    "enable": true,
    "provider": {
      "name": "wrongProvider",
      "oapServer": "localhost:xxx"
    }
  }
}`
)

// TestParseFunctionContext tests and verifies the function that parses the function Context
func TestParseFunctionContext(t *testing.T) {
	_, err := GetOpenFunctionContext()
	if !strings.Contains(err.Error(), "env FUNC_CONTEXT not found") {
		t.Fatal("Error parse function context")
	}

	// test `podName`, `podNamespace` field
	if err := os.Setenv(PodNameEnvName, "test-pod"); err == nil {
		if err := os.Setenv(PodNamespaceEnvName, "test"); err == nil {
			if err := os.Setenv(functionContextEnvName, funcCtxWithKnativeRuntime); err == nil {
				if ctx, err := GetOpenFunctionContext(); err != nil {
					t.Fatalf("Error parse function context: %s", err.Error())
				} else {
					if ctx.podName != "test-pod" {
						t.Fatal("Error parse function context: failed to parse pod name")
					}
					if ctx.podNamespace != "test" {
						t.Fatal("Error parse function context: failed to parse pod namespace")
					}
				}
			} else {
				t.Fatal("Error set function context env")
			}
		} else {
			t.Fatal("Error set pod namespace env")
		}
	} else {
		t.Fatal("Error set pod name env")
	}

	// test `runtime` field
	if err := os.Setenv(functionContextEnvName, baseFuncCtx); err == nil {
		if _, err := GetOpenFunctionContext(); err == nil || !strings.Contains(err.Error(), "invalid runtime") {
			t.Fatal("Error parse function context")
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(functionContextEnvName, funcCtxWithWrongRuntime); err == nil {
		if _, err := GetOpenFunctionContext(); err == nil || !strings.Contains(err.Error(), "invalid runtime") {
			t.Fatal("Error parse function context: failed to parse runtime")
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(functionContextEnvName, funcCtxWithKnativeRuntime); err == nil {
		if ctx, err := GetOpenFunctionContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if ctx.Runtime != Knative {
				t.Fatal("Error parse function context: failed to parse runtime")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(functionContextEnvName, funcCtxWithAsyncRuntime); err == nil {
		if ctx, err := GetOpenFunctionContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if ctx.Runtime != Async {
				t.Fatal("Error parse function context: failed to parse runtime")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	// test `port` field
	if err := os.Setenv(functionContextEnvName, funcCtxWithAsyncRuntime); err == nil {
		if ctx, err := GetOpenFunctionContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if ctx.Port != defaultPort {
				t.Fatal("Error parse function context: failed to parse port")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(functionContextEnvName, funcCtxWithCustomPort); err == nil {
		if ctx, err := GetOpenFunctionContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if ctx.Port != "12345" {
				t.Fatal("Error parse function context: failed to parse port")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(functionContextEnvName, funcCtxWithWrongPort); err == nil {
		if _, err := GetOpenFunctionContext(); err == nil || !strings.Contains(err.Error(), "error parsing port") {
			t.Fatal("Error parse function context: failed to parse port")
		}
	} else {
		t.Fatal("Error set function context env")
	}

	// test `prePlugins`, `postPlugins`, `pluginsTracing` fields
	if err := os.Setenv(functionContextEnvName, funcCtxWithPlugins); err == nil {
		if ctx, err := GetOpenFunctionContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if !(ctx.PrePlugins != nil && len(ctx.PrePlugins) == 3 && ctx.PrePlugins[2] == "plgC") {
				t.Fatal("Error parse function context: failed to parse pre plugins")
			}

			if !(ctx.PostPlugins != nil && len(ctx.PostPlugins) == 2 && ctx.PostPlugins[0] == "plgC") {
				t.Fatal("Error parse function context: failed to parse post plugins")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(functionContextEnvName, funcCtxWithTracingCfg); err == nil {
		if ctx, err := GetOpenFunctionContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if !(ctx.PluginsTracing != nil &&
				ctx.PluginsTracing.Provider.Name == TracingProviderSkywalking &&
				ctx.PluginsTracing.Tags != nil && ctx.PluginsTracing.Tags["layer"] == "faas" &&
				ctx.PluginsTracing.Tags["instance"] == ctx.podName && ctx.PluginsTracing.Tags["namespace"] == ctx.podNamespace &&
				ctx.PluginsTracing.Baggage != nil && ctx.PluginsTracing.Baggage["key"] == "sw8-correlation") {
				t.Fatal("Error parse function context: failed to parse tracing config")
			}

			if !(ctx.PrePlugins[len(ctx.PrePlugins)-1] == TracingProviderSkywalking && ctx.PostPlugins[0] == TracingProviderSkywalking) {
				t.Fatal("Error parse function context: failed to register tracing plugin")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(functionContextEnvName, funcCtxWithWrongTracingCfg); err == nil {
		if _, err := GetOpenFunctionContext(); err == nil || !strings.Contains(err.Error(), "the tracing plugin is enabled, but its configuration is incorrect") {
			t.Fatal("Error parse function context: failed to parse tracing config")
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(functionContextEnvName, funcCtxWithWrongTracingCfgProvider); err == nil {
		if _, err := GetOpenFunctionContext(); err == nil || !strings.Contains(err.Error(), "invalid tracing provider name") {
			t.Fatal("Error parse function context: failed to parse tracing config")
		}
	} else {
		t.Fatal("Error set function context env")
	}
}
