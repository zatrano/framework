package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Resolve picks one release. It does not enable or import the package.
func (idx Index) Resolve(q Query) (Result, error) {
	p, ok := idx.Lookup(q.Name)
	if !ok {
		return Result{}, fmt.Errorf("registry: unknown package %q", q.Name)
	}
	if k := strings.ToLower(strings.TrimSpace(q.Kind)); k != "" && p.Kind != k {
		return Result{}, fmt.Errorf("registry: %q kind is %q, not %q", p.Name, p.Kind, k)
	}
	want := strings.TrimSpace(q.Version)
	if want == "" || strings.EqualFold(want, "latest") {
		rel, err := p.latestCompatible(q.Framework)
		if err != nil {
			return Result{}, err
		}
		return Result{Package: p, Release: rel}, nil
	}
	if strings.EqualFold(want, ChannelMain) {
		rel, ok := p.channel(ChannelMain)
		if !ok {
			return Result{}, fmt.Errorf("registry: %q has no channel %s", p.Name, ChannelMain)
		}
		if !releaseOK(q.Framework, rel.FrameworkMin) {
			return Result{}, fmt.Errorf("registry: %q@main needs framework >= %s", p.Name, rel.FrameworkMin)
		}
		return Result{Package: p, Release: rel}, nil
	}
	want = normalizeVersion(want)
	for _, r := range p.Releases {
		if r.Version == "" {
			continue
		}
		if normalizeVersion(r.Version) != want {
			continue
		}
		if !releaseOK(q.Framework, r.FrameworkMin) {
			return Result{}, fmt.Errorf("registry: %q@%s needs framework >= %s", p.Name, r.Version, r.FrameworkMin)
		}
		return Result{Package: p, Release: r}, nil
	}
	return Result{}, fmt.Errorf("registry: %q has no version %s", p.Name, want)
}

func releaseOK(framework, min string) bool {
	if strings.TrimSpace(framework) == "" {
		return true
	}
	return MeetsFrameworkMin(framework, min)
}

func (p Package) channel(name string) (Release, bool) {
	for _, r := range p.Releases {
		if r.Channel == name {
			return r, true
		}
	}
	return Release{}, false
}

func (p Package) latestCompatible(framework string) (Release, error) {
	var best *Release
	for i := range p.Releases {
		r := p.Releases[i]
		if r.Version == "" {
			continue
		}
		if !releaseOK(framework, r.FrameworkMin) {
			continue
		}
		if best == nil || compareSemver(normalizeVersion(r.Version), normalizeVersion(best.Version)) > 0 {
			cp := r
			best = &cp
		}
	}
	if best != nil {
		return *best, nil
	}
	// No compatible tag: fall back to the source channel, not a published version.
	if r, ok := p.channel(ChannelMain); ok && releaseOK(framework, r.FrameworkMin) {
		return r, nil
	}
	return Release{}, fmt.Errorf("registry: %q has no compatible release", p.Name)
}

// Search lists identities. It does not resolve versions.
func (idx Index) Search(f Filter) []Package {
	q := strings.ToLower(strings.TrimSpace(f.Query))
	kind := strings.ToLower(strings.TrimSpace(f.Kind))
	layer := strings.ToLower(strings.TrimSpace(f.Layer))
	out := make([]Package, 0)
	for _, p := range idx.Packages {
		if kind != "" && p.Kind != kind {
			continue
		}
		if layer != "" && p.Layer != layer {
			continue
		}
		if f.Heavy != nil && p.Heavy != *f.Heavy {
			continue
		}
		if q != "" && !strings.Contains(p.Name, q) && !strings.Contains(strings.ToLower(p.Description), q) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// VerifyDigest reports whether digest matches SHA-256 of manifest bytes.
// An empty digest is not checked (v1 optional integrity).
func VerifyDigest(digest string, manifestBytes []byte) error {
	digest = strings.ToLower(strings.TrimSpace(digest))
	if digest == "" {
		return nil
	}
	sum := sha256.Sum256(manifestBytes)
	got := hex.EncodeToString(sum[:])
	if got != digest {
		return fmt.Errorf("registry: digest mismatch")
	}
	return nil
}

// DigestSHA256 returns the hex SHA-256 of b.
func DigestSHA256(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
