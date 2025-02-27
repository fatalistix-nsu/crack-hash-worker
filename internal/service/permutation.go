package service

import "strings"

type PermutationGenerator struct {
	current  []uint64
	alphabet []rune
	id       uint64
	size     uint64
}

func NewPermutationGenerator(alphabet string, n, size uint64) *PermutationGenerator {
	wordLen := countWordLen(alphabet, n)
	current := nthCombination(alphabet, n, wordLen)

	return &PermutationGenerator{
		alphabet: []rune(alphabet),
		current:  current,
		id:       0,
		size:     size,
	}
}

func countWordLen(alphabet string, n uint64) uint64 {
	base := uint64(len(alphabet))
	sum := uint64(0)
	length := uint64(1)
	power := base

	for {
		sum += power
		if n < sum {
			return length
		}
		length++
		power *= base
	}
}

func nthCombination(alphabet string, n, length uint64) []uint64 {
	base := uint64(len(alphabet))
	n -= sumOfPowers(base, length-1)
	result := make([]uint64, length)

	for i := length; i > 0; i-- {
		result[i-1] = n % base
		n /= base
	}

	return result
}

func sumOfPowers(base, maxExp uint64) uint64 {
	sum := uint64(0)
	power := base
	for i := uint64(0); i < maxExp; i++ {
		sum += power
		power *= base
	}
	return sum
}

func (g *PermutationGenerator) Next() string {
	result := g.makeResult()

	lastIdx := len(g.current) - 1
	alphabetLen := uint64(len(g.alphabet))

	last := g.current[lastIdx]
	last++
	if last == alphabetLen {
		g.current[lastIdx] = 0

		i := lastIdx - 1
		for ; i >= 0; i-- {
			value := g.current[i]
			value++
			if value != alphabetLen {
				g.current[i] = value
				break
			}
			g.current[i] = 0
		}

		if i < 0 {
			g.current = append(g.current, 0)
		}
	} else {
		g.current[len(g.current)-1] = last
	}

	g.id++
	return result
}

func (g *PermutationGenerator) makeResult() string {
	builder := strings.Builder{}
	for i := 0; i < len(g.current); i++ {
		builder.WriteRune(g.alphabet[g.current[i]])
	}

	return builder.String()
}

func (g *PermutationGenerator) HasNext() bool {
	return g.id < g.size
}
