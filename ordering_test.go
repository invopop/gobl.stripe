package goblstripe_test

import (
	"testing"

	goblstripe "github.com/invopop/gobl.stripe"
	"github.com/invopop/gobl/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderingPeriod(t *testing.T) {
	s := validStripeInvoice()
	gi, err := goblstripe.FromInvoice(s, uuid.MustParse(namespace))
	require.NoError(t, err)

	assert.Equal(t, "2025-01-08", gi.Lines[0].Item.Meta[goblstripe.MetaKeyDateFrom])
	assert.Equal(t, "2025-02-08", gi.Lines[0].Item.Meta[goblstripe.MetaKeyDateTo])
}
