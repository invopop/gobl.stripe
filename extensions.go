package goblstripe

import (
	"strings"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

const (
	customDataItemExt = "gobl-item-"
	// Deprecated: the "gobl-customer-" prefix approach for mapping Stripe metadata to
	// GOBL extensions will be removed in a future version. Stripe metadata is now mapped
	// directly to the party's Meta field; any specific mappings should be handled on the
	// client side.
	customDataCustomerExt = "gobl-customer-"
)

// newExtensionsWithPrefix checks if the key starts with the provided prefix and returns a
// tax.Extensions object with the key and value if it does.
func newExtensionsWithPrefix(metadata map[string]string, prefix string) tax.Extensions {
	extensions := tax.Extensions{}
	for key, value := range metadata {
		if strings.HasPrefix(key, prefix) {
			key = strings.TrimPrefix(key, prefix)
			extensions[cbc.Key(key)] = cbc.Code(value)
		}
	}
	return extensions
}

// newMeta converts a Stripe metadata map into a GOBL cbc.Meta map,
// normalizing keys to valid cbc.Key format.
func newMeta(metadata map[string]string) cbc.Meta {
	meta := make(cbc.Meta)
	for key, value := range metadata {
		k := normalizeMetaKey(key)
		if k != "" {
			meta[cbc.Key(k)] = value
		}
	}
	return meta
}

// normalizeMetaKey converts an arbitrary string key into a valid cbc.Key
// format: lowercase, underscores replaced with hyphens, invalid characters
// stripped, and leading/trailing hyphens trimmed.
func normalizeMetaKey(key string) string {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "_", "-")
	var b strings.Builder
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '+' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-+")
}
