package tuf

// This package defines gittuf's take on TUF metadata. There are some minor
// changes, such as the addition of `custom` to delegation entries. Some of it,
// however, is inspired by or cloned from the go-tuf implementation.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/secure-systems-lab/go-securesystemslib/cjson"
)

const specVersion = "1.0"

var (
	ErrTargetsNotEmpty = errors.New("`targets` field in gittuf Targets metadata must be empty")
)

// Key defines the structure for how public keys are stored in TUF metadata.
type Key struct {
	KeyType             string   `json:"keytype"`
	Scheme              string   `json:"scheme"`
	KeyVal              KeyVal   `json:"keyval"`
	KeyIDHashAlgorithms []string `json:"keyid_hash_algorithms"`
	keyID               string
}

// KeyVal contains a `Public` field that records the public key value.
type KeyVal struct {
	Public string `json:"public"`
}

// NewKey creates a new instance of Key for the values passed in.
func NewKey(keyType string, scheme string, keyVal KeyVal, keyIDHashAlgorithms []string) (*Key, error) {
	key := &Key{
		KeyType:             keyType,
		Scheme:              scheme,
		KeyVal:              keyVal,
		KeyIDHashAlgorithms: keyIDHashAlgorithms,
	}
	keyID, err := calculateKeyID(key)
	key.keyID = keyID
	return key, err
}

// LoadKeyFromBytes returns a pointer to a Key instance created from the
// contents of the bytes. The key contents are expected to be in the custom
// securesystemslib format.
func LoadKeyFromBytes(contents []byte) (*Key, error) {
	// FIXME: this assumes keys are stored in securesystemslib format.
	var key *Key
	if err := json.Unmarshal(contents, &key); err != nil {
		return nil, err
	}

	keyID, err := calculateKeyID(key)
	if err != nil {
		return nil, err
	}
	key.keyID = keyID

	return key, nil
}

// ID returns the key ID.
func (k *Key) ID() string {
	return k.keyID
}

func calculateKeyID(k *Key) (string, error) {
	// Modified version of go-tuf's implementation to use a single Key ID.
	canonical, err := cjson.EncodeCanonical(k)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

// Role records common characteristics recorded in a role entry in Root metadata
// and in a delegation entry.
type Role struct {
	KeyIDs    []string `json:"keyids"`
	Threshold int      `json:"threshold"`
}

// RootMetadata defines the schema of TUF's Root role.
type RootMetadata struct {
	Type               string          `json:"type"`
	SpecVersion        string          `json:"spec_version"`
	ConsistentSnapshot bool            `json:"consistent_snapshot"` // TODO: how do we handle this?
	Version            int             `json:"version"`
	Expires            string          `json:"expires"`
	Keys               map[string]Key  `json:"keys"`
	Roles              map[string]Role `json:"roles"`
}

// NewRootMetadata returns a new instance of RootMetadata.
func NewRootMetadata() *RootMetadata {
	return &RootMetadata{
		Type:               "root",
		SpecVersion:        specVersion,
		ConsistentSnapshot: true,
	}
}

// SetVersion sets the version of the RootMetadata to the value passed in.
func (r *RootMetadata) SetVersion(version int) {
	r.Version = version
}

// SetExpires sets the expiry date of the RootMetadata to the value passed in.
func (r *RootMetadata) SetExpires(expires string) {
	r.Expires = expires
}

// AddKey adds a key to the RootMetadata instance.
func (r *RootMetadata) AddKey(key Key) {
	if r.Keys == nil {
		r.Keys = map[string]Key{}
	}

	r.Keys[key.keyID] = key
}

// AddRole adds a role object and associates it with roleName in the
// RootMetadata instance.
func (r *RootMetadata) AddRole(roleName string, role Role) {
	if r.Roles == nil {
		r.Roles = map[string]Role{}
	}

	r.Roles[roleName] = role
}

// TargetsMetadata defines the schema of TUF's Targets role.
type TargetsMetadata struct {
	Type        string         `json:"type"`
	SpecVersion string         `json:"spec_version"`
	Version     int            `json:"version"`
	Expires     string         `json:"expires"`
	Targets     map[string]any `json:"targets"`
	Delegations *Delegations   `json:"delegations"`
}

// NewTargetsMetadata returns a new instance of TargetsMetadata.
func NewTargetsMetadata() *TargetsMetadata {
	return &TargetsMetadata{
		Type:        "targets",
		SpecVersion: specVersion,
	}
}

// SetVersion sets the version of the TargetsMetadata to the value passed in.
func (t *TargetsMetadata) SetVersion(version int) {
	t.Version = version
}

// SetExpires sets the expiry date of the TargetsMetadata to the value passed
// in.
func (t *TargetsMetadata) SetExpires(expires string) {
	t.Expires = expires
}

// Validate ensures the instance of TargetsMetadata matches gittuf expectations.
func (t *TargetsMetadata) Validate() error {
	if len(t.Targets) != 0 {
		return ErrTargetsNotEmpty
	}
	return nil
}

// Delegations defines the schema for specifying delegations in TUF's Targets
// metadata.
type Delegations struct {
	Keys  map[string]Key `json:"keys"`
	Roles []Delegation   `json:"roles"`
}

// AddKey adds a delegations key.
func (d *Delegations) AddKey(key Key) {
	if d.Keys == nil {
		d.Keys = map[string]Key{}
	}

	d.Keys[key.keyID] = key
}

// AddDelegation adds a new delegation.
func (d *Delegations) AddDelegation(delegation Delegation) {
	if d.Roles == nil {
		d.Roles = []Delegation{}
	}

	d.Roles = append(d.Roles, delegation)
}

// Delegation defines the schema for a single delegation entry. It differs from
// the standard TUF schema by allowing a `custom` field to record details
// pertaining to the delegation.
type Delegation struct {
	Name        string           `json:"name"`
	Paths       []string         `json:"paths"`
	Terminating bool             `json:"terminating"`
	Custom      *json.RawMessage `json:"custom,omitempty"`
	Role
}
