package signer

import "fmt"

// bech32Decode decodes a bech32 or bech32m encoded string and returns the
// human-readable part and the data bytes (5-bit groups converted to 8-bit).
func bech32Decode(encoded string) (string, []byte, error) {
	// Find the separator
	sepIdx := -1
	for i := len(encoded) - 1; i >= 0; i-- {
		if encoded[i] == '1' {
			sepIdx = i
			break
		}
	}
	if sepIdx < 1 || sepIdx+7 > len(encoded) {
		return "", nil, fmt.Errorf("invalid bech32 string")
	}

	hrp := encoded[:sepIdx]
	dataStr := encoded[sepIdx+1:]

	// Decode data characters
	data := make([]byte, len(dataStr))
	for i, c := range dataStr {
		idx, ok := bech32CharsetRev(byte(c))
		if !ok {
			return "", nil, fmt.Errorf("invalid bech32 character: %c", c)
		}
		data[i] = idx
	}

	// Verify checksum (we accept both bech32 and bech32m)
	if !bech32VerifyChecksum(hrp, data) {
		return "", nil, fmt.Errorf("invalid bech32 checksum")
	}

	// Strip checksum (last 6 characters)
	data = data[:len(data)-6]

	// Convert 5-bit groups to 8-bit bytes
	converted, err := convertBits(data, 5, 8, false)
	if err != nil {
		return "", nil, err
	}

	return hrp, converted, nil
}

func bech32CharsetRev(c byte) (byte, bool) {
	const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	// Case-insensitive
	if c >= 'A' && c <= 'Z' {
		c = c - 'A' + 'a'
	}
	for i := 0; i < len(charset); i++ {
		if charset[i] == c {
			return byte(i), true
		}
	}
	return 0, false
}

func bech32Polymod(values []byte) uint32 {
	gen := [5]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := uint32(1)
	for _, v := range values {
		b := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ uint32(v)
		for i := 0; i < 5; i++ {
			if (b>>uint(i))&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}

func bech32HrpExpand(hrp string) []byte {
	ret := make([]byte, 0, len(hrp)*2+1)
	for _, c := range hrp {
		ret = append(ret, byte(c>>5))
	}
	ret = append(ret, 0)
	for _, c := range hrp {
		ret = append(ret, byte(c&31))
	}
	return ret
}

func bech32VerifyChecksum(hrp string, data []byte) bool {
	values := append(bech32HrpExpand(hrp), data...)
	p := bech32Polymod(values)
	// bech32: polymod == 1, bech32m: polymod == 0x2bc830a3
	return p == 1 || p == 0x2bc830a3
}

func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	acc := uint32(0)
	bits := uint(0)
	var ret []byte
	maxv := uint32((1 << toBits) - 1)

	for _, v := range data {
		acc = (acc << fromBits) | uint32(v)
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			ret = append(ret, byte((acc>>bits)&maxv))
		}
	}

	if pad {
		if bits > 0 {
			ret = append(ret, byte((acc<<(toBits-bits))&maxv))
		}
	} else {
		if bits >= fromBits {
			return nil, fmt.Errorf("invalid padding")
		}
		if (acc<<(toBits-bits))&maxv != 0 {
			return nil, fmt.Errorf("non-zero padding")
		}
	}

	return ret, nil
}
