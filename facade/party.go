package facade

import "github.com/tx3-lang/go-sdk/signer"

// Party represents a named participant in a transaction.
// A party is either address-only (provides an address for resolution)
// or signer-capable (provides both an address and signing ability).
type Party struct {
	address   string
	signer    signer.Signer
	isSigner  bool
}

// AddressParty creates a read-only party that provides only an address.
func AddressParty(address string) Party {
	return Party{address: address}
}

// SignerParty creates a signer party. The address is read from the signer.
func SignerParty(s signer.Signer) Party {
	return Party{
		address:  s.Address(),
		signer:   s,
		isSigner: true,
	}
}

// partyAddress returns the address for this party.
func (p Party) partyAddress() string {
	return p.address
}
