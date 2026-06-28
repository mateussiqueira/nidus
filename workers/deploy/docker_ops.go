package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/prometheus/client_golang/prometheus"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerOps wraps the Docker SDK client for build/run/inspect operations.
//
// It uses the native Docker Engine API instead of shelling out to the `docker`
// CLI, which gives us streaming builds, finer-grained control and lower overhead.
// This is the heart of the deploy worker — performance-critical.
type DockerOps struct {
	cli        *client.Client
	buildkitOK bool
}

// NewDockerOps creates a DockerOps using the standard Docker env discovery.
// Returns an error if Docker is unreachable; callers may fall back to CLI.
func NewDockerOps() (*DockerOps, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client init: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("docker ping: %w", err)
	}

	return &DockerOps{cli: cli, buildkitOK: true}, nil
}

// BuildImage builds a Docker image from a context directory using the native
// Docker Engine API with a tar-based context stream. Build output is forwarded
// to logFn line by line, and cache hits are tracked via Prometheus metrics.
func (d *DockerOps) BuildImage(ctx context.Context, tags []string, dockerfilePath, contextDir string, buildArgs map[string]*string, logFn func(string)) error {
	buildStart := time.Now()
	defer func() { buildDuration.Observe(time.Since(buildStart).Seconds()) }()

	// Create a tar archive of the build context
	tarBuf, err := tarContext(contextDir)
	if err != nil {
		return fmt.Errorf("tar context: %w", err)
	}

	// BuildKit is requested via header; Docker negotiates if available
	options := image.BuildOptions{
		Dockerfile:  dockerfilePath,
		Tags:        tags,
		BuildArgs:   buildArgs,
		Remove:      true,
		ForceRemove: false,
		// BuildKit is enabled through the daemon; we observe CACHED lines below.
	}

	resp, err := d.cli.ImageBuild(ctx, tarBuf, options)
	if err != nil {
		buildCacheMisses.Inc()
		return fmt.Errorf("image build: %w", err)
	}
	defer resp.Body.Close()

	// Stream JSON-formatted build output from the daemon
	return readBuildStream(resp.Body, logFn)
}

// readBuildStream parses the Docker daemon's JSON-line build output and
// forwards human-readable progress lines to logFn while tracking cache hits.
func readBuildStream(body io.ReadCloser, logFn func(string)) error {
	dec := newJSONDecoder(body)
	for {
		msg, err := dec.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read build stream: %w", err)
		}

		text := strings.TrimSpace(msg.Stream)
		if text != "" {
			logFn(strings.TrimRight(text, "\n"))
			if strings.Contains(text, "CACHED") || strings.Contains(text, "Using cache") {
				buildCacheHits.Inc()
			}
		}
		if msg.Error != "" {
			buildCacheMisses.Inc()
			return fmt.Errorf("build error: %s", msg.Error)
		}
	}
}

// CreateAndStartContainer creates a container from an image, applies resource
// limits (memory + CPU), injects env vars, publishes the exposed port and starts it.
// Returns the host-side port assigned by Docker.
func (d *DockerOps) CreateAndStartContainer(ctx context.Context, opts ContainerOpts) (string, error) {
	containerStart := time.Now()
	defer func() {
		_ = opts.LogFn(fmt.Sprintf("⏱️  Container setup took %v", time.Since(containerStart)))
	}()

	// Remove any existing container with the same name
	_ = d.cli.ContainerRemove(ctx, opts.Name, container.RemoveOptions{Force: true})

	// Resource limits keep noisy neighbours in check on small boxes (512MB).
	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		PortBindings: map[natPort][]natPortBinding{
			natPort(fmt.Sprintf("%d/tcp", opts.ExposedPort)): {{HostIP: "0.0.0.0"}},
		},
		Resources: container.Resources{
			Memory:   opts.MemoryBytes,
			NanoCPUs: opts.NanoCPUs,
		},
	}

	envSlice := make([]string, 0, len(opts.Env))
	for k, v := range opts.Env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
		opts.LogFn(fmt.Sprintf("🔧 ENV: %s=%s", k, maskEnvVar(v)))
	}

	containerConfig := &container.Config{
		Image: opts.Image,
		Env:   envSlice,
		ExposedPorts: map[string]struct{}{
			fmt.Sprintf("%d/tcp", opts.ExposedPort): {},
		},
	}

	createResp, err := d.cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		&network.NetworkingConfig{},
		&specs.Platform{},
		opts.Name,
	)
	if err != nil {
		return "", fmt.Errorf("container create: %w", err)
	}

	if err := d.cli.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("container start: %w", err)
	}

	// Resolve the host-side port that Docker assigned
	return d.GetContainerPort(ctx, opts.Name, opts.ExposedPort)
}

// GetContainerPort returns the host port bound to the given container port.
func (d *DockerOps) GetContainerPort(ctx context.Context, name string, containerPort int) (string, error) {
	inspect, err := d.cli.ContainerInspect(ctx, name)
	if err != nil {
		return "", fmt.Errorf("inspect port: %w", err)
	}

	bindings, ok := inspect.NetworkSettings.Ports[natPort(fmt.Sprintf("%d/tcp", containerPort))]
	if !ok || len(bindings) == 0 {
		return "", fmt.Errorf("no port binding for %d", containerPort)
	}
	return bindings[0].HostPort, nil
}

// WaitForContainer polls the container state until it is running or the attempts run out.
func (d *DockerOps) WaitForContainer(ctx context.Context, name string, attempts int, interval time.Duration, logFn func(string)) (bool, error) {
	for i := 0; i < attempts; i++ {
		time.Sleep(interval)
		inspect, err := d.cli.ContainerInspect(ctx, name)
		if err != nil {
			continue
		}
		if inspect.State != nil && inspect.State.Running {
			logFn("✅ Container is running")
			return true, nil
		}
		logFn(fmt.Sprintf("⏳ Waiting for container... (%d/%d)", i+1, attempts))
	}
	return false, nil
}

// ContainerLogs returns the last N lines of a container's logs (for debug on failure).
func (d *DockerOps) ContainerLogs(ctx context.Context, name string, lines int) (string, error) {
	reader, err := d.cli.ContainerLogs(ctx, name, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", lines),
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ContainerOpts configures CreateAndStartContainer.
type ContainerOpts struct {
	Name         string
	Image        string
	ExposedPort  int
	Env          map[string]string
	MemoryBytes  int64
	NanoCPUs     int64
	LogFn        func(string)
}

// tarContext builds an in-memory tar archive of the build context directory.
// It skips .git internals to keep the context small and builds fast.
func tarContext(dir string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip .git internals but allow the repo files
		rel, _ := filepath.Rel(dir, path)
		if strings.HasPrefix(rel, ".git/") || rel == ".git" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// parseMemory parses strings like "512m", "2g" into bytes.
func parseMemory(s string) int64 {
	n, err := units.RAMInBytes(s)
	if err != nil {
		return 512 * 1024 * 1024 // default 512MB
	}
	return n
}

// A small alias to avoid importing moby/types everywhere in call sites below.
type buildMsg struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

// newJSONDecoder reads Docker's newline-delimited JSON build output.
func newJSONDecoder(r io.Reader) *jsonLineDecoder {
	return &jsonLineDecoder{r: bufioReader(r)}
}

type jsonLineDecoder struct {
	r *bufioReaderT
}

func (d *jsonLineDecoder) Next() (buildMsg, error) {
	var msg buildMsg
	line, err := d.r.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return msg, err
	}
	if len(line) == 0 {
		return msg, io.EOF
	}
	_ = jsonUnmarshal(line, &msg)
	return msg, nil
}

// maskEnvVar masks sensitive values for logging.
func maskEnvVar(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

// metrics guard so unused imports don't break the build in trimmed configs.
var _ = prometheus.NewCounter
