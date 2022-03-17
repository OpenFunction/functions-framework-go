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
    "enabled": true,
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
    "enabled": true,
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
    "enabled": true,
    "provider": {
      "name": "wrongProvider",
      "oapServer": "localhost:xxx"
    }
  }
}`
)

// TestParseFunctionContext tests and verifies the function that parses the function FunctionContext
func TestParseFunctionContext(t *testing.T) {
	_, err := GetRuntimeContext()
	if !strings.Contains(err.Error(), "env FUNC_CONTEXT not found") {
		t.Fatal("Error parse function context")
	}

	// test `podName`, `podNamespace` field
	if err := os.Setenv(PodNameEnvName, "test-pod"); err == nil {
		if err := os.Setenv(PodNamespaceEnvName, "test"); err == nil {
			if err := os.Setenv(FunctionContextEnvName, funcCtxWithKnativeRuntime); err == nil {
				if ctx, err := GetRuntimeContext(); err != nil {
					t.Fatalf("Error parse function context: %s", err.Error())
				} else {
					if ctx.GetPodName() != "test-pod" {
						t.Fatal("Error parse function context: failed to parse pod name")
					}
					if ctx.GetPodNamespace() != "test" {
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
	if err := os.Setenv(FunctionContextEnvName, baseFuncCtx); err == nil {
		if _, err := GetRuntimeContext(); err == nil || !strings.Contains(err.Error(), "invalid runtime") {
			t.Fatal("Error parse function context")
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtxWithWrongRuntime); err == nil {
		if _, err := GetRuntimeContext(); err == nil || !strings.Contains(err.Error(), "invalid runtime") {
			t.Fatal("Error parse function context: failed to parse runtime")
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtxWithKnativeRuntime); err == nil {
		if ctx, err := GetRuntimeContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if ctx.GetRuntime() != Knative {
				t.Fatal("Error parse function context: failed to parse runtime")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtxWithAsyncRuntime); err == nil {
		if ctx, err := GetRuntimeContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if ctx.GetRuntime() != Async {
				t.Fatal("Error parse function context: failed to parse runtime")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	// test `port` field
	if err := os.Setenv(FunctionContextEnvName, funcCtxWithAsyncRuntime); err == nil {
		if ctx, err := GetRuntimeContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if ctx.GetPort() != defaultPort {
				t.Fatal("Error parse function context: failed to parse port")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtxWithCustomPort); err == nil {
		if ctx, err := GetRuntimeContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if ctx.GetPort() != "12345" {
				t.Fatal("Error parse function context: failed to parse port")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtxWithWrongPort); err == nil {
		if _, err := GetRuntimeContext(); err == nil || !strings.Contains(err.Error(), "error parsing port") {
			t.Fatal("Error parse function context: failed to parse port")
		}
	} else {
		t.Fatal("Error set function context env")
	}

	// test `inputs`, `outputs` fields
	if err := os.Setenv(FunctionContextEnvName, funcCtx); err == nil {
		if ctx, err := GetRuntimeContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			// test `inputs`
			if !ctx.HasInputs() || len(ctx.GetInputs()) != 2 {
				t.Fatal("Error parse function context: failed to parse inputs")
			}
			if cron, exist := ctx.GetInputs()["cron"]; exist {
				if cron.Uri == "cron_input" && cron.ComponentType == "bindings.cron" {

				} else {
					t.Fatal("Error parse function context: failed to parse input cron")
				}
			}

			// test `outputs`
			if !ctx.HasOutputs() || len(ctx.GetOutputs()) != 3 {
				t.Fatal("Error parse function context: failed to parse outputs")
			}
			if echo, exist := ctx.GetOutputs()["echo"]; exist {
				if path, exist := echo.Metadata["path"]; exist && echo.Uri == "echo" && path == "echo" {

				} else {
					t.Fatal("Error parse function context: failed to parse output echo")
				}
			}
		}
	}

	// test `prePlugins`, `postPlugins`, `pluginsTracing` fields
	if err := os.Setenv(FunctionContextEnvName, funcCtxWithPlugins); err == nil {
		if ctx, err := GetRuntimeContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if !(ctx.GetPrePlugins() != nil && len(ctx.GetPrePlugins()) == 3 && ctx.GetPrePlugins()[2] == "plgC") {
				t.Fatal("Error parse function context: failed to parse pre plugins")
			}

			if !(ctx.GetPostPlugins() != nil && len(ctx.GetPostPlugins()) == 2 && ctx.GetPostPlugins()[0] == "plgC") {
				t.Fatal("Error parse function context: failed to parse post plugins")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtxWithTracingCfg); err == nil {
		if ctx, err := GetRuntimeContext(); err != nil {
			t.Fatalf("Error parse function context: %s", err.Error())
		} else {
			if !(ctx.GetPluginsTracingCfg() != nil &&
				ctx.GetPluginsTracingCfg().ProviderName() == TracingProviderSkywalking &&
				ctx.GetPluginsTracingCfg().GetTags()["layer"] == "faas" &&
				ctx.GetPluginsTracingCfg().GetTags()["instance"] == ctx.GetPodName() && ctx.GetPluginsTracingCfg().GetTags()["namespace"] == ctx.GetPodNamespace() &&
				ctx.GetPluginsTracingCfg().GetBaggage()["key"] == "sw8-correlation") {
				t.Fatal("Error parse function context: failed to parse tracing config")
			}

			if !(ctx.GetPrePlugins()[len(ctx.GetPrePlugins())-1] == TracingProviderSkywalking && ctx.GetPostPlugins()[0] == TracingProviderSkywalking) {
				t.Fatal("Error parse function context: failed to register tracing plugin")
			}
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtxWithWrongTracingCfg); err == nil {
		if _, err := GetRuntimeContext(); err == nil || !strings.Contains(err.Error(), "the tracing plugin is enabled, but its configuration is incorrect") {
			t.Fatal("Error parse function context: failed to parse tracing config")
		}
	} else {
		t.Fatal("Error set function context env")
	}

	if err := os.Setenv(FunctionContextEnvName, funcCtxWithWrongTracingCfgProvider); err == nil {
		if _, err := GetRuntimeContext(); err == nil || !strings.Contains(err.Error(), "invalid tracing provider name") {
			t.Fatal("Error parse function context: failed to parse tracing config")
		}
	} else {
		t.Fatal("Error set function context env")
	}
}
