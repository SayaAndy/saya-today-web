package factgiver

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/b2"
)

type FactGiver struct {
	b2Client      *b2.B2Client
	cache         map[string][]string
	langs         []string
	factsFileName string
	nlRe          *regexp.Regexp
	randGen       *rand.Rand
}

func NewFactGiver(cfg *config.FactGiverConfig, langs []string) (*FactGiver, error) {
	b2Client, err := b2.NewB2Client(&cfg.Storage.Config)
	if err != nil {
		return nil, fmt.Errorf("fail to init b2 client for a new fact giver: %s", err.Error())
	}

	factGiver := &FactGiver{
		b2Client:      b2Client,
		cache:         make(map[string][]string, len(langs)),
		langs:         langs,
		factsFileName: cfg.FactsFileName,
		nlRe:          regexp.MustCompile(`\r?\n`),
		randGen:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if err = factGiver.initCache(); err != nil {
		return nil, fmt.Errorf("fail to init cache for a new fact giver: %s", err.Error())
	}

	return factGiver, nil
}

func (g *FactGiver) Give(lang string) [3]string {
	factSlice := make([]string, len(g.cache[lang]))
	copy(factSlice, g.cache[lang])
	g.randGen.Shuffle(len(factSlice), func(i, j int) {
		factSlice[i], factSlice[j] = factSlice[j], factSlice[i]
	})
	return [3]string{factSlice[0], factSlice[1], factSlice[2]}
}

func (g *FactGiver) initCache() error {
	for _, lang := range g.langs {
		localFacts := strings.Replace(g.factsFileName, "*", lang, 1)
		factsContentBytes, err := g.b2Client.ReadAll(localFacts)
		if err != nil {
			return fmt.Errorf("fail to read '%s' facts file: %s", lang, err.Error())
		}
		factsContent := string(factsContentBytes)
		g.cache[lang] = g.nlRe.Split(factsContent, -1)
		if g.cache[lang][len(g.cache[lang])-1] == "" {
			g.cache[lang] = g.cache[lang][:len(g.cache[lang])-1]
		}
	}
	return nil
}
