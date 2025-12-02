package goblstripe

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
	"github.com/stripe/stripe-go/v84"
)

// newDelivery creates a delivery object from an invoice.
func newDelivery(doc *stripe.Invoice) *bill.DeliveryDetails {
	if doc.ShippingDetails != nil {
		return FromShippingDetailsToDeliveryDetails(doc.ShippingDetails)
	}

	if doc.CustomerShipping != nil {
		return FromShippingDetailsToDeliveryDetails(doc.CustomerShipping)
	}

	// If no shipping details are provided, return nil
	return nil
}

// FromShippingDetailsToDeliveryDetails converts a stripe shipping details object into a GOBL delivery object.
func FromShippingDetailsToDeliveryDetails(shipping *stripe.ShippingDetails) *bill.DeliveryDetails {
	receiver := newReceiver(shipping)
	if receiver.Validate() != nil {
		return nil
	}
	return &bill.DeliveryDetails{
		Receiver: receiver,
	}
}

// newReceiver creates a receiver object from shipping details.
func newReceiver(shipping *stripe.ShippingDetails) *org.Party {
	return &org.Party{
		Name:       shipping.Name,
		Addresses:  []*org.Address{FromAddress(shipping.Address)},
		Telephones: []*org.Telephone{FromTelephone(shipping.Phone)},
	}
}
