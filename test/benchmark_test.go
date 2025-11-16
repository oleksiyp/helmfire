package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oleksiyp/helmfire/pkg/helmstate"
	"github.com/oleksiyp/helmfire/pkg/substitute"
	"github.com/oleksiyp/helmfire/pkg/sync"
	"go.uber.org/zap"
)

// BenchmarkHelmfileLoad benchmarks loading a helmfile
func BenchmarkHelmfileLoad(b *testing.B) {
	tmpDir := b.TempDir()
	helmfilePath := filepath.Join(tmpDir, "helmfile.yaml")

	helmfileContent := `
repositories:
  - name: bitnami
    url: https://charts.bitnami.com/bitnami

releases:
  - name: nginx
    namespace: default
    chart: bitnami/nginx
    version: 13.2.0
  - name: postgres
    namespace: default
    chart: bitnami/postgresql
    version: 12.0.0
  - name: redis
    namespace: default
    chart: bitnami/redis
    version: 17.0.0
`

	if err := os.WriteFile(helmfilePath, []byte(helmfileContent), 0o644); err != nil {
		b.Fatalf("failed to write helmfile: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager := helmstate.NewManager(helmfilePath, "")
		if err := manager.Load(); err != nil {
			b.Fatalf("Load failed: %v", err)
		}
	}
}

// BenchmarkSubstitutionManager benchmarks substitution operations
func BenchmarkSubstitutionManager(b *testing.B) {
	manager := substitute.NewManager()

	b.Run("AddChartSubstitution", func(b *testing.B) {
		tmpDir := b.TempDir()
		chartPath := filepath.Join(tmpDir, "chart")
		if err := os.MkdirAll(chartPath, 0o755); err != nil {
			b.Fatalf("failed to create chart dir: %v", err)
		}

		chartYAML := `apiVersion: v2
name: test-chart
version: 1.0.0
`
		if err := os.WriteFile(filepath.Join(chartPath, "Chart.yaml"), []byte(chartYAML), 0o644); err != nil {
			b.Fatalf("failed to write Chart.yaml: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.AddChartSubstitution("bitnami/nginx", chartPath)
		}
	})

	b.Run("AddImageSubstitution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.AddImageSubstitution("nginx:1.21", "nginx:1.22")
		}
	})

	b.Run("GetChartPath", func(b *testing.B) {
		tmpDir := b.TempDir()
		chartPath := filepath.Join(tmpDir, "chart")
		if err := os.MkdirAll(chartPath, 0o755); err != nil {
			b.Fatalf("failed to create chart dir: %v", err)
		}

		chartYAML := `apiVersion: v2
name: test-chart
version: 1.0.0
`
		if err := os.WriteFile(filepath.Join(chartPath, "Chart.yaml"), []byte(chartYAML), 0o644); err != nil {
			b.Fatalf("failed to write Chart.yaml: %v", err)
		}

		_ = manager.AddChartSubstitution("bitnami/nginx", chartPath)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = manager.GetChartPath("bitnami/nginx")
		}
	})

	b.Run("GetImageReplacement", func(b *testing.B) {
		_ = manager.AddImageSubstitution("nginx:1.21", "nginx:1.22")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = manager.GetImageReplacement("nginx:1.21")
		}
	})
}

// BenchmarkFilterReleases benchmarks release filtering
func BenchmarkFilterReleases(b *testing.B) {
	tmpDir := b.TempDir()
	helmfilePath := filepath.Join(tmpDir, "helmfile.yaml")

	helmfileContent := `
releases:
  - name: nginx-1
    chart: bitnami/nginx
    labels:
      app: web
      tier: frontend
  - name: nginx-2
    chart: bitnami/nginx
    labels:
      app: web
      tier: frontend
  - name: postgres
    chart: bitnami/postgresql
    labels:
      app: db
      tier: backend
  - name: redis
    chart: bitnami/redis
    labels:
      app: cache
      tier: backend
  - name: mongodb
    chart: bitnami/mongodb
    labels:
      app: db
      tier: backend
`

	if err := os.WriteFile(helmfilePath, []byte(helmfileContent), 0o644); err != nil {
		b.Fatalf("failed to write helmfile: %v", err)
	}

	manager := helmstate.NewManager(helmfilePath, "")
	if err := manager.Load(); err != nil {
		b.Fatalf("Load failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.FilterReleases(map[string]string{"tier": "backend"})
	}
}

// BenchmarkCreateImagePostRenderer benchmarks post-renderer creation
func BenchmarkCreateImagePostRenderer(b *testing.B) {
	logger := zap.NewNop()
	sub := substitute.NewManager()

	// Add multiple image substitutions
	_ = sub.AddImageSubstitution("nginx:1.21", "nginx:1.22")
	_ = sub.AddImageSubstitution("postgres:15", "postgres:16")
	_ = sub.AddImageSubstitution("redis:7", "redis:7-alpine")

	executor := sync.NewExecutor(logger, sub)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scriptPath, err := executor.CreateImagePostRendererForBenchmark()
		if err != nil {
			b.Fatalf("createImagePostRenderer failed: %v", err)
		}
		os.Remove(scriptPath)
	}
}

// BenchmarkLoadValuesFile benchmarks values file loading
func BenchmarkLoadValuesFile(b *testing.B) {
	tmpDir := b.TempDir()
	valuesPath := filepath.Join(tmpDir, "values.yaml")

	valuesContent := `
replicaCount: 3

image:
  repository: nginx
  tag: 1.21
  pullPolicy: IfNotPresent

service:
  type: LoadBalancer
  port: 80
  targetPort: 8080

ingress:
  enabled: true
  annotations:
    kubernetes.io/ingress.class: nginx
  hosts:
    - host: example.com
      paths:
        - path: /
          pathType: Prefix

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi
`

	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0o644); err != nil {
		b.Fatalf("failed to write values file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sync.LoadValuesFile(valuesPath)
		if err != nil {
			b.Fatalf("LoadValuesFile failed: %v", err)
		}
	}
}
