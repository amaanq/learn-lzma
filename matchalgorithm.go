package lzma

import "errors"

type MatchAlgorithm byte

const (
	HashTable4 MatchAlgorithm = iota
	BinaryTree
)

var maStrings = map[MatchAlgorithm]string{
	HashTable4: "HashTable4",
	BinaryTree: "BinaryTree",
}

func (a MatchAlgorithm) String() string {
	if s, ok := maStrings[a]; ok {
		return s
	}
	return "unknown"
}

var errUnsupportedMatchAlgorithm = errors.New("lzma: unsupported match algorithm value")

func (a MatchAlgorithm) verify() error {
	if _, ok := maStrings[a]; !ok {
		return errUnsupportedMatchAlgorithm
	}
	return nil
}

func (a MatchAlgorithm) new(dictCap int) (matcher, error) {
	switch a {
	case HashTable4:
		return newHashTable(dictCap, 4)
	case BinaryTree:
		return newBinTree(dictCap)
	}
	return nil, errUnsupportedMatchAlgorithm
}
