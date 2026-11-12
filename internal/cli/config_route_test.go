package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aimuzov/happ-cli/internal/store"
)

func TestConfigCommandUsesRoutingConfigAndWritesToOutput(t *testing.T) {
	dir := t.TempDir()
	setTestHome(t, dir)
	yaml := "connect:\n  default-outbound: direct\n  include:\n    - chosen.example\n  exclude:\n    - bypass.example\n  dns-proxy: false\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	deps := withDeps()
	ms := deps.Store.(*mockStore)
	ms.subs = []store.SubEntry{{Name: "vpn", Links: []string{"vless://11111111-1111-4111-8111-111111111111@vpn.example:443#test"}}}
	ms.active = "vpn"
	cmd := newConfigCmd(deps)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, expected := range []string{`"outboundTag": "direct"`, `"domain:chosen.example"`, `"domain:bypass.example"`, `"network": "tcp,udp"`} {
		if !strings.Contains(got, expected) {
			t.Errorf("generated config missing %s", expected)
		}
	}
	if strings.Contains(got, `"dns"`) {
		t.Error("dns-proxy=false should omit DNS config")
	}
}
