package audit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type adoptionProvenance struct {
	SchemaVersion int `json:"schemaVersion"`
	Product       struct {
		Repository             string `json:"repository"`
		AuthoritativeRef       string `json:"authoritativeRef"`
		Revision               string `json:"revision"`
		RemoteVerifiedAt       string `json:"remoteVerifiedAt"`
		GeneratedClientTreeOID string `json:"generatedClientTreeGitOid"`
		OpenAPIBlobOID         string `json:"openapiBlobGitOid"`
		OpenAPISHA256          string `json:"openapiSha256"`
	} `json:"product"`
	FctlSDK struct {
		ModulePath string `json:"modulePath"`
		Repository string `json:"repository"`
		Revision   string `json:"revision"`
		SDKPath    string `json:"sdkPath"`
		SDKNarHash string `json:"sdkNarHash"`
		WITPath    string `json:"witPath"`
		WITSHA256  string `json:"witSha256"`
		Wrapper    string `json:"wrapper"`
	} `json:"fctlSdk"`
	RFC0011 struct {
		Status   string `json:"status"`
		Path     string `json:"path"`
		SHA256   string `json:"sha256"`
		Revision string `json:"fctlRevision"`
	} `json:"rfc0011"`
	Evidence struct {
		ExecutableOperations     int  `json:"executableOperations"`
		AdmissionBlocked         int  `json:"admissionBlockedOperations"`
		ReleaseGapOperations     int  `json:"releaseGapOperations"`
		CommonPayloadBytes       int  `json:"commonPayloadBytes"`
		GeneratedMethodBoundary  bool `json:"generatedMethodBoundary"`
		AmbientAuthDisabled      bool `json:"ambientAuthDisabled"`
		AutomaticRetriesDisabled bool `json:"automaticRetriesDisabled"`
		GetWithBodyPreserved     bool `json:"getWithBodyPreserved"`
		LosslessWideIntegers     bool `json:"losslessWideIntegers"`
	} `json:"evidence"`
}

type sdkSourceLock struct {
	SchemaVersion int    `json:"schemaVersion"`
	ModulePath    string `json:"modulePath"`
	Repository    string `json:"repository"`
	Commit        string `json:"commit"`
	SDKPath       string `json:"sdkPath"`
	SDKNarHash    string `json:"sdkNarHash"`
	WITPath       string `json:"witPath"`
	WITSHA256     string `json:"witSha256"`
}

func commandOutput(t *testing.T, directory string, arguments ...string) string {
	t.Helper()
	command := exec.Command(arguments[0], arguments[1:]...)
	command.Dir = directory
	output, err := command.Output()
	if err != nil {
		t.Fatalf("%s: %v", strings.Join(arguments, " "), err)
	}
	return strings.TrimSpace(string(output))
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:])
}

func TestGeneratedClientAdoptionProvenanceIsExact(t *testing.T) {
	encoded, err := os.ReadFile(filepath.Join("..", "docs", "generated-client-provenance.json"))
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	var provenance adoptionProvenance
	if err := json.Unmarshal(encoded, &provenance); err != nil {
		t.Fatalf("decode provenance: %v", err)
	}

	moduleRoot := filepath.Clean("..")
	productRoot := filepath.Clean(filepath.Join(moduleRoot, "..", ".."))
	if provenance.SchemaVersion != 1 || provenance.Product.Repository != "https://github.com/formancehq/payments.git" || provenance.Product.AuthoritativeRef != "refs/heads/main" {
		t.Fatalf("unexpected product provenance identity: %#v", provenance.Product)
	}
	if _, err := time.Parse(time.RFC3339, provenance.Product.RemoteVerifiedAt); err != nil {
		t.Fatalf("remote verification timestamp is not RFC3339: %q", provenance.Product.RemoteVerifiedAt)
	}
	if got := commandOutput(t, productRoot, "git", "config", "--get", "remote.origin.url"); got != provenance.Product.Repository {
		t.Fatalf("origin URL = %q, want %q", got, provenance.Product.Repository)
	}
	if got := commandOutput(t, productRoot, "git", "rev-parse", "origin/main"); got != provenance.Product.Revision {
		t.Fatalf("origin/main = %s, want %s", got, provenance.Product.Revision)
	}
	if got := commandOutput(t, productRoot, "git", "rev-parse", provenance.Product.Revision+":pkg/client"); got != provenance.Product.GeneratedClientTreeOID {
		t.Fatalf("generated client tree = %s, want %s", got, provenance.Product.GeneratedClientTreeOID)
	}
	if got := commandOutput(t, productRoot, "git", "rev-parse", provenance.Product.Revision+":openapi.yaml"); got != provenance.Product.OpenAPIBlobOID {
		t.Fatalf("OpenAPI blob = %s, want %s", got, provenance.Product.OpenAPIBlobOID)
	}
	if got := fileSHA256(t, filepath.Join(productRoot, "openapi.yaml")); got != provenance.Product.OpenAPISHA256 {
		t.Fatalf("OpenAPI SHA-256 = %s, want %s", got, provenance.Product.OpenAPISHA256)
	}
	diff := exec.Command("git", "diff", "--quiet", provenance.Product.Revision, "--", "pkg/client", "openapi.yaml")
	diff.Dir = productRoot
	if err := diff.Run(); err != nil {
		t.Fatalf("working generated client or OpenAPI differs from recorded product revision: %v", err)
	}

	if provenance.FctlSDK.Revision != "545521bfa222250af6b4419b194c7967cded0379" || provenance.RFC0011.Revision != provenance.FctlSDK.Revision {
		t.Fatalf("unexpected fctl/RFC revision: fctl=%s rfc=%s", provenance.FctlSDK.Revision, provenance.RFC0011.Revision)
	}
	lockEncoded, err := os.ReadFile(filepath.Join(moduleRoot, "fctl-sdk.lock.json"))
	if err != nil {
		t.Fatalf("read fctl SDK lock: %v", err)
	}
	var lock sdkSourceLock
	if err := json.Unmarshal(lockEncoded, &lock); err != nil {
		t.Fatalf("decode fctl SDK lock: %v", err)
	}
	if lock.SchemaVersion != 1 || provenance.FctlSDK.ModulePath != lock.ModulePath || provenance.FctlSDK.Repository != lock.Repository ||
		provenance.FctlSDK.Revision != lock.Commit || provenance.FctlSDK.SDKPath != lock.SDKPath || provenance.FctlSDK.SDKNarHash != lock.SDKNarHash ||
		provenance.FctlSDK.WITPath != lock.WITPath || provenance.FctlSDK.WITSHA256 != lock.WITSHA256 || provenance.FctlSDK.Wrapper != "./scripts/with-fctl-sdk.sh" {
		t.Fatalf("fctl SDK provenance differs from source lock: provenance=%#v lock=%#v", provenance.FctlSDK, lock)
	}
	if filepath.IsAbs(lock.SDKPath) || filepath.IsAbs(lock.WITPath) {
		t.Fatalf("fctl SDK lock paths must be repository-relative: %#v", lock)
	}
	sdkDirectory := commandOutput(t, moduleRoot, "go", "list", "-m", "-f", "{{.Dir}}", lock.ModulePath)
	fctlRoot := filepath.Clean(filepath.Join(sdkDirectory, "..", ".."))
	if got, want := filepath.Clean(sdkDirectory), filepath.Join(fctlRoot, lock.SDKPath); got != want {
		t.Fatalf("effective SDK directory = %q, want locked repository-relative %q", got, want)
	}
	if got := fileSHA256(t, filepath.Join(fctlRoot, lock.WITPath)); got != lock.WITSHA256 {
		t.Fatalf("fctl SDK WIT SHA-256 = %s, want %s", got, lock.WITSHA256)
	}
	if filepath.IsAbs(provenance.RFC0011.Path) || filepath.Clean(provenance.RFC0011.Path) != "docs/rfcs/0011-generated-product-client-host-adapters.md" {
		t.Fatalf("RFC path must be the canonical repository-relative path, got %q", provenance.RFC0011.Path)
	}
	if provenance.RFC0011.Status != "Validated" || provenance.RFC0011.SHA256 != "e12b973a0cbb214550c36316afc2fc691f6d5f89ab236b0f3ce01de5e76b6ffe" {
		t.Fatalf("RFC 0011 proof does not match the pinned source metadata")
	}
	if provenance.Evidence.ExecutableOperations != 44 || provenance.Evidence.AdmissionBlocked != 0 || provenance.Evidence.ReleaseGapOperations != 3 ||
		provenance.Evidence.CommonPayloadBytes != 4*1024*1024 ||
		!provenance.Evidence.GeneratedMethodBoundary || !provenance.Evidence.AmbientAuthDisabled ||
		!provenance.Evidence.AutomaticRetriesDisabled || !provenance.Evidence.GetWithBodyPreserved ||
		!provenance.Evidence.LosslessWideIntegers {
		t.Fatalf("incomplete generated-client adoption evidence: %#v", provenance.Evidence)
	}
}
