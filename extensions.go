package goblstripe

import (
	"strings"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

const (
	customDataItemExt     = "gobl-item-"
	customDataCustomerExt = "gobl-customer-"
	customDataVATExt      = "gobl-line-vat-"
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
