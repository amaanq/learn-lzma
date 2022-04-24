package hash

type Roller interface {
	Len() int
	RollByte(x byte) uint64
}

// Hashes computes all hash values for the array p. Note that the state of the
// roller is changed.
func Hashes(r Roller, p []byte) []uint64 {
	n := r.Len()
	if len(p) < n {
		return nil
	}
	h := make([]uint64, len(p)-n+1)
	for i := 0; i < n-1; i++ {
		r.RollByte(p[i])
	}
	for i := range h {
		h[i] = r.RollByte(p[i+n-1])
	}
	return h
}
