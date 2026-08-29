// Copyright(c) 2026 The Rainway AI Gateway (壬远AI网关) Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cleanup_test

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rainway-ai-gateway/conf-agent/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const module = "mod_demo"

func TestCleanup_KeepsVersionKeepCountVersions(t *testing.T) {
	tmpDir := t.TempDir()
	confDir := filepath.Join(tmpDir, module)

	confServer := testutil.NewMockConfigServer(t, module)
	bfeServer := testutil.NewMockBFEServer(t, module)

	confURL, _ := url.Parse(confServer.URL())
	bfeURL, _ := url.Parse(bfeServer.URL())

	runner := testutil.StartReloader(t, module, confDir, confURL, bfeURL,
		testutil.WithVersionKeepCount(2))

	// wait for first reload
	runner.WaitForReload(t, bfeServer, 1, 5*time.Second)
	target1 := testutil.CurrentTarget(t, confDir)
	t.Logf("first reload target: %s", target1)

	// update version to trigger second reload
	confServer.SetVersion("20260101120001")
	runner.WaitForReload(t, bfeServer, 2, 5*time.Second)
	target2 := testutil.CurrentTarget(t, confDir)
	t.Logf("second reload target: %s", target2)

	assert.NotEqual(t, target1, target2, "target should change after version update")

	// update version to trigger third reload
	confServer.SetVersion("20260101120002")
	runner.WaitForReload(t, bfeServer, 3, 5*time.Second)
	target3 := testutil.CurrentTarget(t, confDir)
	t.Logf("third reload target: %s", target3)

	assert.NotEqual(t, target2, target3, "target should change after version update")

	runner.Stop()

	// verify only 2 version directories exist (current + 1 previous)
	versionDirs := listVersionDirs(t, tmpDir, module)
	assert.Len(t, versionDirs, 2, "should keep exactly VersionKeepCount directories")
	assert.Contains(t, versionDirs, filepath.Base(target3), "current target should be kept")
	assert.Contains(t, versionDirs, filepath.Base(target2), "previous target should be kept")
	assert.NotContains(t, versionDirs, filepath.Base(target1), "oldest target should be removed")

	// verify BFE received reload requests with correct paths
	assert.GreaterOrEqual(t, bfeServer.ReloadCount(), 3, "BFE should receive at least 3 reload requests")
	assert.True(t, strings.Contains(bfeServer.LastReloadPath(), "20260101120002"), "last reload should use latest version")
}

func TestCleanup_KeepsCurrentWhenBFEReloadFails(t *testing.T) {
	tmpDir := t.TempDir()
	confDir := filepath.Join(tmpDir, module)

	confServer := testutil.NewMockConfigServer(t, module)
	bfeServer := testutil.NewMockBFEServer(t, module)
	bfeServer.SetFail(true)

	confURL, _ := url.Parse(confServer.URL())
	bfeURL, _ := url.Parse(bfeServer.URL())

	runner := testutil.StartReloader(t, module, confDir, confURL, bfeURL,
		testutil.WithVersionKeepCount(2))

	// wait a bit; BFE reload fails, so symlink should not be updated
	time.Sleep(500 * time.Millisecond)

	// ConfDir should not exist as a symlink because UpdateDefaultConfDir was not called
	_, err := os.Lstat(confDir)
	require.True(t, os.IsNotExist(err), "ConfDir should not exist when BFE reload fails")

	// tmp dir should exist because StoreFile2TmpDir was called
	tmpDirs := listVersionDirs(t, tmpDir, module)
	assert.Len(t, tmpDirs, 1, "one tmp dir should be created even when BFE reload fails")

	runner.Stop()
}

func listVersionDirs(t *testing.T, parentDir, module string) []string {
	t.Helper()

	entries, err := os.ReadDir(parentDir)
	require.NoError(t, err)

	var dirs []string
	prefix := module + "_"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && !strings.HasSuffix(name, ".backup") {
			markerFile := filepath.Join(parentDir, name, ".conf-agent-version")
			if _, err := os.Stat(markerFile); err == nil {
				dirs = append(dirs, name)
			}
		}
	}

	return dirs
}
