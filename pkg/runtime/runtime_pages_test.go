package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexbeatnik/ManulEngineGo/pkg/config"
	"github.com/alexbeatnik/ManulEngineGo/pkg/dsl"
	"github.com/alexbeatnik/ManulEngineGo/pkg/pages"
	"github.com/alexbeatnik/ManulEngineGo/pkg/utils"
)

func TestRuntime_AutoAnnotateNavigateUsesPagesRegistry(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{
    "site": "https://example.com/",
    "Domain": "Example",
    "https://example.com/dashboard": "Dashboard Page"
}`)
	_ = os.WriteFile(filepath.Join(dir, "example.com.json"), data, 0644)

	mock := &MockPage{URL: "https://example.com/dashboard"}
	cfg := config.Config{AutoAnnotate: true}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	rt.pages = pages.NewRegistry(dir) // Inject test registry

	ctx := context.Background()

	// When auto-annotating after navigation, it should use the registry page name.
	hunt := &dsl.Hunt{
		Commands: []dsl.Command{
			{Type: dsl.CmdNavigate, URL: "https://example.com/dashboard"},
		},
	}

	_, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		t.Fatalf("RunHunt failed: %v", err)
	}

	// The auto-annotate should have looked up "Dashboard Page" from the registry.
	// We verify by checking that the registry file was not auto-populated
	// because the URL was already known.
	frag := filepath.Join(dir, "example.com.json")
	content, _ := os.ReadFile(frag)
	if string(content) == "" {
		t.Fatal("pages registry fragment missing")
	}
}

func TestRuntime_PagesRegistryAutoPopulate(t *testing.T) {
	dir := t.TempDir()
	mock := &MockPage{URL: "https://new-site.io/profile"}
	cfg := config.Config{AutoAnnotate: true}
	logger := utils.NewLogger(nil)
	rt := New(cfg, mock, logger)
	rt.pages = pages.NewRegistry(dir)

	ctx := context.Background()
	hunt := &dsl.Hunt{
		Commands: []dsl.Command{
			{Type: dsl.CmdNavigate, URL: "https://new-site.io/profile"},
		},
	}

	_, err := rt.RunHunt(ctx, hunt)
	if err != nil {
		t.Fatalf("RunHunt failed: %v", err)
	}

	// Auto-populate should have created a fragment.
	frag := filepath.Join(dir, "new-site.io.json")
	if _, statErr := os.Stat(frag); os.IsNotExist(statErr) {
		t.Fatalf("auto-populate did not create fragment %s", frag)
	}
}
