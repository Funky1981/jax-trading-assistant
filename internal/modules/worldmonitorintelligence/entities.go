package worldmonitorintelligence

import (
	"sort"
	"strings"
)

const EntityExtractionAlgorithmV1 = "jax.world_monitor_entity_extraction/v1"

type EntityKind string

const (
	EntityIssuer       EntityKind = "ISSUER"
	EntityOrganization EntityKind = "ORGANIZATION"
	EntityCompany      EntityKind = "COMPANY"
)

type Entity struct {
	ID          string
	Kind        EntityKind
	Canonical   string
	AliasesSeen []string
}

type Geography struct {
	ID          string
	Canonical   string
	AliasesSeen []string
}

type EntityExtraction struct {
	Algorithm   string
	Entities    []Entity
	Geographies []Geography
	Unknowns    []string
}

type entityLexiconEntry struct {
	id      string
	name    string
	kind    EntityKind
	aliases []string
}

type geographyLexiconEntry struct {
	id      string
	name    string
	aliases []string
}

var entityLexicon = []entityLexiconEntry{
	{id: "ent_apple", name: "Apple", kind: EntityCompany, aliases: []string{"apple", "aapl"}},
	{id: "ent_federal_reserve", name: "Federal Reserve", kind: EntityIssuer, aliases: []string{"federal reserve", "fed"}},
	{id: "ent_microsoft", name: "Microsoft", kind: EntityCompany, aliases: []string{"microsoft", "msft"}},
	{id: "ent_nvidia", name: "NVIDIA", kind: EntityCompany, aliases: []string{"nvidia", "nvda"}},
	{id: "ent_opec", name: "OPEC", kind: EntityOrganization, aliases: []string{"opec"}},
	{id: "ent_sec", name: "U.S. Securities and Exchange Commission", kind: EntityIssuer, aliases: []string{"securities and exchange commission", "sec"}},
	{id: "ent_treasury", name: "U.S. Treasury", kind: EntityIssuer, aliases: []string{"us treasury", "u.s. treasury", "treasury department"}},
}

var geographyLexicon = []geographyLexiconEntry{
	{id: "geo_china", name: "China", aliases: []string{"china", "prc"}},
	{id: "geo_europe", name: "Europe", aliases: []string{"europe", "european union", "eu"}},
	{id: "geo_middle_east", name: "Middle East", aliases: []string{"middle east"}},
	{id: "geo_russia", name: "Russia", aliases: []string{"russia"}},
	{id: "geo_taiwan", name: "Taiwan", aliases: []string{"taiwan"}},
	{id: "geo_ukraine", name: "Ukraine", aliases: []string{"ukraine"}},
	{id: "geo_united_kingdom", name: "United Kingdom", aliases: []string{"united kingdom", "uk"}},
	{id: "geo_united_states", name: "United States", aliases: []string{"united states", "us", "u.s.", "america"}},
}

// ExtractEntities uses a versioned, deterministic lexicon. It deliberately
// returns unknowns instead of inventing entities from model output or text
// similarity. The result is a link layer, not a trade instruction.
func ExtractEntities(cluster EventCluster) EntityExtraction {
	text := normalizeEntityText(cluster)
	entities := make([]Entity, 0)
	for _, entry := range entityLexicon {
		aliases := matchingAliases(text, entry.aliases)
		if len(aliases) > 0 {
			sort.Strings(aliases)
			entities = append(entities, Entity{ID: entry.id, Kind: entry.kind, Canonical: entry.name, AliasesSeen: aliases})
		}
	}
	geographies := make([]Geography, 0)
	for _, entry := range geographyLexicon {
		aliases := matchingAliases(text, entry.aliases)
		if len(aliases) > 0 {
			sort.Strings(aliases)
			geographies = append(geographies, Geography{ID: entry.id, Canonical: entry.name, AliasesSeen: aliases})
		}
	}
	sort.Slice(entities, func(left, right int) bool { return entities[left].ID < entities[right].ID })
	sort.Slice(geographies, func(left, right int) bool { return geographies[left].ID < geographies[right].ID })
	unknowns := append([]string(nil), cluster.Unknowns...)
	if len(entities) == 0 {
		unknowns = append(unknowns, "no supported issuer or organization/entity match")
	}
	if len(geographies) == 0 {
		unknowns = append(unknowns, "no supported geography match")
	}
	sort.Strings(unknowns)
	return EntityExtraction{Algorithm: EntityExtractionAlgorithmV1, Entities: entities, Geographies: geographies, Unknowns: uniqueStrings(unknowns)}
}

func normalizeEntityText(cluster EventCluster) string {
	parts := make([]string, 0, len(cluster.Members)*2)
	for _, member := range cluster.Members {
		parts = append(parts, member.Title, member.Summary)
	}
	text := strings.ToLower(strings.Join(parts, " "))
	text = strings.NewReplacer(".", " ", ",", " ", "-", " ", "_", " ", "'", " ").Replace(text)
	return " " + strings.Join(strings.Fields(text), " ") + " "
}

func matchingAliases(text string, aliases []string) []string {
	matched := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		normalized := strings.ToLower(strings.TrimSpace(alias))
		normalized = strings.NewReplacer(".", " ", "-", " ").Replace(normalized)
		normalized = strings.Join(strings.Fields(normalized), " ")
		if normalized != "" && strings.Contains(text, " "+normalized+" ") {
			matched = append(matched, alias)
		}
	}
	return matched
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
