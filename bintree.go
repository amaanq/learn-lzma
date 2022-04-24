package lzma

import "errors"

type node struct {
	x uint32 // search
	p uint32 // parent
	l uint32 // left
	r uint32 // right
}

const wordLen = 4

type binTree struct {
	dict  *encoderDict
	node  []node
	hoff  int64
	front uint32
	root  uint32
	x     uint32
	data  []byte
}

const null uint32 = 1<<32 - 1

func newBinTree(capacity int) (*binTree, error) {
	if capacity < 1 {
		return nil, errors.New("newBinTree: capacity must be larger than zero")
	}
	if int64(capacity) >= int64(null) {
		return nil, errors.New("newBinTree: capacity must less 2^{32}-1")
	}

	return &binTree{
		node: make([]node, capacity),
		hoff: -int64(wordLen),
		root: null,
		data: make([]byte, maxMatchLen),
	}, nil
}

func (t *binTree) SetDict(d *encoderDict) { t.dict = d }

func (t *binTree) WriteByte(c byte) error {
	t.x = (t.x << 8) | uint32(c)
	t.hoff++
	if t.hoff < 0 {
		return nil
	}
	v := t.front
	if int64(v) < t.hoff {
		t.remove(v)
	}
	t.node[v].x = t.x
	t.add(v)
	t.front++
	if int64(t.front) >= int64(len(t.node)) {
		t.front = 0
	}
	return nil
}

func (t *binTree) Write(p []byte) (int, error) {
	for _, c := range p {
		t.WriteByte(c)
	}
	return len(p), nil
}

func (t *binTree) add(v uint32) {
	vn := &t.node[v]

	vn.l, vn.r = null, null

	if t.root == null {
		t.root = v
		vn.p = null
		return
	}
	x := vn.x
	p := t.root
	for {
		pn := &t.node[p]
		if x <= pn.x {
			if pn.l == null {
				pn.l = v
				vn.p = p
				return
			}
			p = pn.l
		} else {
			if pn.r == null {
				pn.r = v
				vn.p = p
				return
			}
			p = pn.r
		}
	}
}

func (t *binTree) parent(v uint32) (uint32, *uint32) {
	if t.root == v {
		return null, &t.root
	}
	p := t.node[v].p
	var ptr *uint32
	if t.node[p].l == v {
		ptr = &t.node[p].l
	} else {
		ptr = &t.node[p].r
	}
	return p, ptr
}

func (t *binTree) remove(v uint32) {
	vn := &t.node[v]
	p, ptr := t.parent(v)
	l, r := vn.l, vn.r
	if l == null {
		*ptr = r
		if r != null {
			t.node[r].p = p
		}
		return
	}
	if r == null {
		*ptr = l
		t.node[l].p = p
		return
	}

	un := &t.node[l]
	ur := un.r
	if ur == null {
		un.r = r
		t.node[r].p = l
		un.p = p
		*ptr = l
		return
	}
	var u uint32
	for {
		u = ur
		ur = t.node[u].r
		if ur == null {
			break
		}
	}
	un = &t.node[u]
	ul := un.l
	up := un.p
	t.node[up].r = ul
	if ul != null {
		t.node[ul].p = up
	}

	un.l, un.r = l, r
	t.node[l].p = u
	t.node[r].p = u
	*ptr = u
	un.p = p
}

func (t *binTree) search(v uint32, x uint32) (uint32, uint32) {
	a, b := null, null
	if v == null {
		return a, b
	}
	for {
		vn := &t.node[v]
		if x <= vn.x {
			if x == vn.x {
				return v, v
			}
			b = v
			if vn.l == null {
				return a, b
			}
			v = vn.l
		} else {
			a = v
			if vn.r == null {
				return a, b
			}
			v = vn.r
		}
	}
}

func (t *binTree) max(v uint32) uint32 {
	if v == null {
		return null
	}
	for {
		r := t.node[v].r
		if r == null {
			return v
		}
		v = r
	}
}

func (t *binTree) pred(v uint32) uint32 {
	if v == null {
		return null
	}
	u := t.max(t.node[v].l)
	if u != null {
		return u
	}
	for {
		p := t.node[v].p
		if p == null {
			return null
		}
		if t.node[p].r == v {
			return p
		}
		v = p
	}
}

func xval(a []byte) uint32 {
	var x uint32
	switch len(a) {
	default:
		x |= uint32(a[3])
		fallthrough
	case 3:
		x |= uint32(a[2]) << 8
		fallthrough
	case 2:
		x |= uint32(a[1]) << 16
	case 1:
		x |= uint32(a[0]) << 24
	}
	return x
}

func (t *binTree) distance(v uint32) int {
	dist := int(t.front) - int(v)
	if dist <= 0 {
		dist += len(t.node)
	}
	return dist
}

type matchParams struct {
	rep [4]uint32
	// length when match will be accepted
	nAccept int
	// nodes to check
	check int
	// finish if length get shorter
	stopShorter bool
}

func (t *binTree) match(m match, distIter func() (int, bool), p matchParams) (match, int, bool) {
	var checked int
	buf := &t.dict.buf
	for {
		if checked >= p.check {
			return m, checked, true
		}
		dist, ok := distIter()
		if !ok {
			return m, checked, false
		}
		checked++
		if m.n > 0 {
			i := buf.rear - dist + m.n - 1
			if i < 0 {
				i += len(buf.data)
			} else if i >= len(buf.data) {
				i -= len(buf.data)
			}
			if buf.data[i] != t.data[m.n-1] {
				if p.stopShorter {
					return m, checked, false
				}
				continue
			}
		}
		n := buf.matchLen(dist, t.data)
		switch n {
		case 0:
			if p.stopShorter {
				return m, checked, false
			}
			continue
		case 1:
			if uint32(dist-minDistance) != p.rep[0] {
				continue
			}
		}
		if n < m.n || (n == m.n && int64(dist) >= m.distance) {
			continue
		}
		m = match{int64(dist), n}
		if n >= p.nAccept {
			return m, checked, true
		}
	}
}

func (t *binTree) NextOp(rep [4]uint32) operation {
	n, _ := t.dict.buf.Peek(t.data[:maxMatchLen])
	if n == 0 {
		panic("no data in buffer")
	}
	t.data = t.data[:n]

	var (
		m                  match
		x, u, v            uint32
		iterPred, iterSucc func() (int, bool)
	)
	p := matchParams{
		rep:     rep,
		nAccept: maxMatchLen,
		check:   32,
	}
	i := 4
	iterSmall := func() (int, bool) {
		i--
		if i <= 0 {
			return 0, false
		}
		return i, true
	}
	m, checked, accepted := t.match(m, iterSmall, p)
	if accepted {
		goto end
	}
	p.check -= checked
	x = xval(t.data)
	u, v = t.search(t.root, x)
	if u == v && len(t.data) == 4 {
		iter := func() (int, bool) {
			if u == null {
				return 0, false
			}
			dist := t.distance(u)
			u, v := t.search(t.node[u].l, x)
			if u != v {
				u = null
			}
			return dist, true
		}
		m, _, _ = t.match(m, iter, p)
		goto end
	}
	p.stopShorter = true
	iterSucc = func() (int, bool) {
		if v == null {
			return 0, false
		}
		dist := t.distance(u)
		u, v = t.search(t.node[u].l, x)
		if u != v {
			u = null
		}
		return dist, true
	}
	m, checked, accepted = t.match(m, iterSucc, p)
	if accepted {
		goto end
	}
	p.check -= checked
	iterPred = func() (int, bool) {
		if u == null {
			return 0, false
		}
		dist := t.distance(u)
		u = t.pred(u)
		return dist, true
	}
	m, _, _ = t.match(m, iterPred, p)
end:
	if m.n == 0 {
		return lit{t.data[0]}
	}
	return m
}
