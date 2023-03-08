package controllers

import (
	"fmt"
	"testing"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

func TestBuildObjectsToApply(t *testing.T) {
	testCases := map[string]struct {
		rrsInput      string
		gvk           schema.GroupVersionKind
		syncNamespace string
		expectedSync  string
	}{
		"minimal": {
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync
spec:
  template:
    metadata: null
`,
			gvk:           rootSyncGVK,
			syncNamespace: rootSyncNamespace,
			expectedSync: `apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  labels:
    gitops.kpt.dev/remoterootsync-name: my-remote-root-sync
    gitops.kpt.dev/remoterootsync-namespace: ""
  name: my-remote-root-sync
  namespace: config-management-system
`,
		},

		"minimal, rootSync type specified": {
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync
spec:
  template:
    type: RootSync
`,
			gvk:           rootSyncGVK,
			syncNamespace: rootSyncNamespace,
			expectedSync: `apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  labels:
    gitops.kpt.dev/remoterootsync-name: my-remote-root-sync
    gitops.kpt.dev/remoterootsync-namespace: ""
  name: my-remote-root-sync
  namespace: config-management-system
`,
		},

		"minimal, repoSync type specified": {
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync
spec:
  template:
    type: RepoSync
`,
			gvk:           repoSyncGVK,
			syncNamespace: "",
			expectedSync: `apiVersion: configsync.gke.io/v1beta1
kind: RepoSync
metadata:
  labels:
    gitops.kpt.dev/remoterootsync-name: my-remote-root-sync
    gitops.kpt.dev/remoterootsync-namespace: ""
  name: my-remote-root-sync
`,
		},
		"with additional labels and annotations": {
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync
spec:
  template:
    metadata:
      labels:
       foo: bar
      annotations:
       abc: def
       efg: hij
`,
			gvk:           rootSyncGVK,
			syncNamespace: rootSyncNamespace,
			expectedSync: `apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  annotations:
    abc: def
    efg: hij
  labels:
    foo: bar
    gitops.kpt.dev/remoterootsync-name: my-remote-root-sync
    gitops.kpt.dev/remoterootsync-namespace: ""
  name: my-remote-root-sync
  namespace: config-management-system
`,
		},
		"with source format": {
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync
spec:
  template:
    spec:
      sourceFormat: unstructured
`,
			gvk:           rootSyncGVK,
			syncNamespace: rootSyncNamespace,
			expectedSync: `apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  labels:
    gitops.kpt.dev/remoterootsync-name: my-remote-root-sync
    gitops.kpt.dev/remoterootsync-namespace: ""
  name: my-remote-root-sync
  namespace: config-management-system
spec:
  sourceFormat: unstructured
`,
		},
		"with git info": {
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync
spec:
  template:
    spec:
      sourceFormat: unstructured
      git:
        repo: blueprints
        branch: main
        dir: namespaces
`,
			gvk:           rootSyncGVK,
			syncNamespace: rootSyncNamespace,
			expectedSync: `apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  labels:
    gitops.kpt.dev/remoterootsync-name: my-remote-root-sync
    gitops.kpt.dev/remoterootsync-namespace: ""
  name: my-remote-root-sync
  namespace: config-management-system
spec:
  git:
    auth: ""
    branch: main
    dir: namespaces
    period: 0s
    repo: blueprints
    secretRef: {}
  sourceFormat: unstructured
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var rrs gitopsv1alpha1.RemoteRootSync
			require.NoError(t, yaml.Unmarshal([]byte(tc.rrsInput), &rrs))

			u, err := BuildObjectsToApply(&rrs, tc.gvk, tc.syncNamespace)
			require.NoError(t, err)

			actual, err := yaml.Marshal(u)
			require.NoError(t, err)

			require.Equal(t, tc.expectedSync, string(actual))
		})
	}
}

func TestGetExternalSyncNamespace(t *testing.T) {
	testCases := map[string]struct {
		rrsInput string
		expected string
	}{
		"empty type": { // should default to RootSync
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync`,
			expected: rootSyncNamespace,
		},
		"RootSync type": {
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync
spec:
  type: RootSync`,
			expected: rootSyncNamespace,
		},
		"RepoSync type": {
			rrsInput: `apiVersion: gitops.kpt.dev/v1alpha1
kind: RemoteRootSync
metadata:
  name: my-remote-root-sync
  namespace: foo
spec:
  type: RepoSync`,
			expected: "foo",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var rrs gitopsv1alpha1.RemoteRootSync
			require.NoError(t, yaml.Unmarshal([]byte(tc.rrsInput), &rrs))
			require.Equal(t, tc.expected, getExternalSyncNamespace(&rrs))
		})
	}
}

func TestGetGvrAndGvk(t *testing.T) {
	testCases := map[string]struct {
		input gitopsv1alpha1.SyncTemplateType

		expectedGVK schema.GroupVersionKind
		expectedGVR schema.GroupVersionResource
		expectedErr error
	}{
		"empty type": { // should default to RootSync
			input: "",

			expectedGVK: rootSyncGVK,
			expectedGVR: rootSyncGVR,
		},
		"RootSync type": {
			input: gitopsv1alpha1.TemplateTypeRootSync,

			expectedGVK: rootSyncGVK,
			expectedGVR: rootSyncGVR,
		},
		"RepoSync type": {
			input: gitopsv1alpha1.TemplateTypeRepoSync,

			expectedGVK: repoSyncGVK,
			expectedGVR: repoSyncGVR,
		},
		"unsupported type": {
			input: "unsupported",

			expectedErr: fmt.Errorf(`invalid sync type "unsupported"`),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actualGVR, actualGVK, actualErr := getGvrAndGvk(tc.input)

			require.Equal(t, tc.expectedGVK, actualGVK)
			require.Equal(t, tc.expectedGVR, actualGVR)
			require.Equal(t, tc.expectedErr, actualErr)
		})
	}
}
