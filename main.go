package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const defaultBaseURL = "https://wol.jw.org/es/wol/b/r4/lp-s/nwtsty"

type chapterFetcher func(ctx context.Context, bookID int, chapter int) ([]byte, string, error)

type Citation struct {
	BookID    int
	BookName  string
	Chapter   int
	Specs     []VerseSpec
	Multiline bool
}

type VerseSpec struct {
	Start int
	End   int
}

type Verse struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type Result struct {
	Reference string  `json:"reference"`
	SourceURL string  `json:"source_url"`
	Verses    []Verse `json:"verses"`
	Text      string  `json:"text"`
}

type book struct {
	ID      int
	Name    string
	Aliases []string
}

var citationPattern = regexp.MustCompile(`^\s*(.+?)\s+([0-9]+)\s*:\s*(.+?)\s*$`)

var books = []book{
	{1, "Génesis", []string{"Génesis", "Gén.", "Gé", "Genesis", "Gen", "Ge"}},
	{2, "Éxodo", []string{"Éxodo", "Éx.", "Éx"}},
	{3, "Levítico", []string{"Levítico", "Lev.", "Le"}},
	{4, "Números", []string{"Números", "Núm.", "Nú"}},
	{5, "Deuteronomio", []string{"Deuteronomio", "Deut.", "Dt"}},
	{6, "Josué", []string{"Josué", "Jos.", "Jos"}},
	{7, "Jueces", []string{"Jueces", "Juec.", "Jue"}},
	{8, "Rut", []string{"Rut"}},
	{9, "1 Samuel", []string{"1 Samuel", "1 Sam.", "1Sa"}},
	{10, "2 Samuel", []string{"2 Samuel", "2 Sam.", "2Sa"}},
	{11, "1 Reyes", []string{"1 Reyes", "1 Rey.", "1Re"}},
	{12, "2 Reyes", []string{"2 Reyes", "2 Rey.", "2Re"}},
	{13, "1 Crónicas", []string{"1 Crónicas", "1 Crón.", "1Cr"}},
	{14, "2 Crónicas", []string{"2 Crónicas", "2 Crón.", "2Cr"}},
	{15, "Esdras", []string{"Esdras", "Esd.", "Esd"}},
	{16, "Nehemías", []string{"Nehemías", "Neh.", "Ne"}},
	{17, "Ester", []string{"Ester", "Est.", "Est"}},
	{18, "Job", []string{"Job"}},
	{19, "Salmos", []string{"Salmos", "Sal.", "Sl"}},
	{20, "Proverbios", []string{"Proverbios", "Prov.", "Pr"}},
	{21, "Eclesiastés", []string{"Eclesiastés", "Ecl.", "Ec"}},
	{22, "El Cantar de los Cantares", []string{"El Cantar de los Cantares", "Cant.", "Can"}},
	{23, "Isaías", []string{"Isaías", "Is.", "Is"}},
	{24, "Jeremías", []string{"Jeremías", "Jer.", "Jer"}},
	{25, "Lamentaciones", []string{"Lamentaciones", "Lam.", "Lam"}},
	{26, "Ezequiel", []string{"Ezequiel", "Ezeq.", "Eze"}},
	{27, "Daniel", []string{"Daniel", "Dan.", "Da"}},
	{28, "Oseas", []string{"Oseas", "Os.", "Os"}},
	{29, "Joel", []string{"Joel", "Joe"}},
	{30, "Amós", []string{"Amós", "Am"}},
	{31, "Abdías", []string{"Abdías", "Abd.", "Abd"}},
	{32, "Jonás", []string{"Jonás", "Jon.", "Jon"}},
	{33, "Miqueas", []string{"Miqueas", "Miq.", "Miq"}},
	{34, "Nahúm", []string{"Nahúm", "Nah.", "Na"}},
	{35, "Habacuc", []string{"Habacuc", "Hab.", "Hab"}},
	{36, "Sofonías", []string{"Sofonías", "Sof.", "Sof"}},
	{37, "Ageo", []string{"Ageo", "Ag"}},
	{38, "Zacarías", []string{"Zacarías", "Zac.", "Zac"}},
	{39, "Malaquías", []string{"Malaquías", "Mal.", "Mal"}},
	{40, "Mateo", []string{"Mateo", "Mat.", "Mt"}},
	{41, "Marcos", []string{"Marcos", "Mar.", "Mr"}},
	{42, "Lucas", []string{"Lucas", "Luc.", "Lu"}},
	{43, "Juan", []string{"Juan", "Jn"}},
	{44, "Hechos", []string{"Hechos", "Hech.", "Hch"}},
	{45, "Romanos", []string{"Romanos", "Rom.", "Ro"}},
	{46, "1 Corintios", []string{"1 Corintios", "1 Cor.", "1Co"}},
	{47, "2 Corintios", []string{"2 Corintios", "2 Cor.", "2Co"}},
	{48, "Gálatas", []string{"Gálatas", "Gál.", "Gál"}},
	{49, "Efesios", []string{"Efesios", "Efes.", "Ef"}},
	{50, "Filipenses", []string{"Filipenses", "Filip.", "Flp"}},
	{51, "Colosenses", []string{"Colosenses", "Col.", "Col"}},
	{52, "1 Tesalonicenses", []string{"1 Tesalonicenses", "1 Tes.", "1Te"}},
	{53, "2 Tesalonicenses", []string{"2 Tesalonicenses", "2 Tes.", "2Te"}},
	{54, "1 Timoteo", []string{"1 Timoteo", "1 Tim.", "1Ti"}},
	{55, "2 Timoteo", []string{"2 Timoteo", "2 Tim.", "2Ti"}},
	{56, "Tito", []string{"Tito", "Tit"}},
	{57, "Filemón", []string{"Filemón", "Filem.", "Flm"}},
	{58, "Hebreos", []string{"Hebreos", "Heb.", "Heb"}},
	{59, "Santiago", []string{"Santiago", "Sant.", "Snt"}},
	{60, "1 Pedro", []string{"1 Pedro", "1 Ped.", "1Pe"}},
	{61, "2 Pedro", []string{"2 Pedro", "2 Ped.", "2Pe"}},
	{62, "1 Juan", []string{"1 Juan", "1Jn"}},
	{63, "2 Juan", []string{"2 Juan", "2Jn"}},
	{64, "3 Juan", []string{"3 Juan", "3Jn"}},
	{65, "Judas", []string{"Judas", "Jud.", "Jud"}},
	{66, "Apocalipsis", []string{"Apocalipsis", "Apoc.", "Ap"}},
}

var bookAliases = buildBookAliases()

func main() {
	client := &http.Client{Timeout: 20 * time.Second}
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, httpChapterFetcher(defaultBaseURL, client)))
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, fetch chapterFetcher) int {
	fs := flag.NewFlagSet("wol-cite", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "emit JSON")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	input := strings.Join(fs.Args(), " ")
	if strings.TrimSpace(input) == "" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "error: reading stdin: %v\n", err)
			return 1
		}
		input = string(data)
	}

	result, err := lookup(context.Background(), input, fetch)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(stderr, "error: writing JSON: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprintln(stdout, result.Text)
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: wol-cite [--json] [cita]")
}

func lookup(ctx context.Context, input string, fetch chapterFetcher) (Result, error) {
	citation, err := parseCitation(input)
	if err != nil {
		return Result{}, err
	}
	if fetch == nil {
		return Result{}, errors.New("missing chapter fetcher")
	}

	body, sourceURL, err := fetch(ctx, citation.BookID, citation.Chapter)
	if err != nil {
		return Result{}, err
	}

	verses, err := extractVerses(body, citation.BookID, citation.Chapter, citation.VerseNumbers())
	if err != nil {
		return Result{}, err
	}

	return Result{
		Reference: citation.Reference(),
		SourceURL: sourceURL,
		Verses:    verses,
		Text:      formatVerses(verses, citation.Multiline),
	}, nil
}

func parseCitation(input string) (Citation, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Citation{}, errors.New("empty citation")
	}

	matches := citationPattern.FindStringSubmatch(input)
	if matches == nil {
		return Citation{}, fmt.Errorf("invalid citation %q; expected Libro Capitulo:Versiculos", input)
	}

	bookName := strings.TrimSpace(matches[1])
	chapter, err := parsePositiveInt(matches[2], "chapter")
	if err != nil {
		return Citation{}, err
	}

	book, ok := bookAliases[normalizeBookKey(bookName)]
	if !ok {
		return Citation{}, fmt.Errorf("unknown book %q", bookName)
	}

	specs, multiline, err := parseVerseSpecs(matches[3])
	if err != nil {
		return Citation{}, err
	}

	return Citation{
		BookID:    book.ID,
		BookName:  book.Name,
		Chapter:   chapter,
		Specs:     specs,
		Multiline: multiline,
	}, nil
}

func parseVerseSpecs(input string) ([]VerseSpec, bool, error) {
	parts := strings.Split(input, ",")
	specs := make([]VerseSpec, 0, len(parts))

	for _, rawPart := range parts {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			return nil, false, fmt.Errorf("invalid verse list %q", input)
		}

		if strings.Count(part, "-") > 0 {
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return nil, false, fmt.Errorf("invalid verse range %q", part)
			}
			start, err := parsePositiveInt(strings.TrimSpace(bounds[0]), "verse")
			if err != nil {
				return nil, false, err
			}
			end, err := parsePositiveInt(strings.TrimSpace(bounds[1]), "verse")
			if err != nil {
				return nil, false, err
			}
			if end < start {
				return nil, false, fmt.Errorf("invalid descending verse range %d-%d", start, end)
			}
			specs = append(specs, VerseSpec{Start: start, End: end})
			continue
		}

		verse, err := parsePositiveInt(part, "verse")
		if err != nil {
			return nil, false, err
		}
		specs = append(specs, VerseSpec{Start: verse, End: verse})
	}

	if len(specs) == 0 {
		return nil, false, errors.New("missing verses")
	}

	return specs, len(parts) > 1, nil
}

func parsePositiveInt(input string, label string) (int, error) {
	number, err := strconv.Atoi(input)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("invalid %s %q", label, input)
	}
	return number, nil
}

func (c Citation) VerseNumbers() []int {
	var numbers []int
	for _, spec := range c.Specs {
		for verse := spec.Start; verse <= spec.End; verse++ {
			numbers = append(numbers, verse)
		}
	}
	return numbers
}

func (c Citation) Reference() string {
	parts := make([]string, 0, len(c.Specs))
	for _, spec := range c.Specs {
		if spec.Start == spec.End {
			parts = append(parts, strconv.Itoa(spec.Start))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", spec.Start, spec.End))
		}
	}
	return fmt.Sprintf("%s %d:%s", c.BookName, c.Chapter, strings.Join(parts, ", "))
}

func httpChapterFetcher(baseURL string, client *http.Client) chapterFetcher {
	return func(ctx context.Context, bookID int, chapter int) ([]byte, string, error) {
		sourceURL := fmt.Sprintf("%s/%d/%d", strings.TrimRight(baseURL, "/"), bookID, chapter)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return nil, sourceURL, err
		}
		req.Header.Set("User-Agent", "wol-cite/0.1")

		resp, err := client.Do(req)
		if err != nil {
			return nil, sourceURL, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, sourceURL, fmt.Errorf("WOL returned %s for %s", resp.Status, sourceURL)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
		if err != nil {
			return nil, sourceURL, err
		}
		return body, sourceURL, nil
	}
}

func extractVerses(body []byte, bookID int, chapter int, requested []int) ([]Verse, error) {
	root, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	requestedSet := make(map[int]bool, len(requested))
	for _, verse := range requested {
		requestedSet[verse] = true
	}

	found := make(map[int][]string, len(requested))
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "v") {
			foundBook, foundChapter, foundVerse, ok := parseVerseID(attr(n, "id"))
			if ok && foundBook == bookID && foundChapter == chapter && requestedSet[foundVerse] {
				fragment := normalizeSpace(verseText(n))
				if fragment != "" {
					found[foundVerse] = append(found[foundVerse], fragment)
				}
			}
			return
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)

	verses := make([]Verse, 0, len(requested))
	for _, verse := range requested {
		fragments := found[verse]
		if len(fragments) == 0 {
			return nil, fmt.Errorf("verse %d not found in WOL chapter", verse)
		}
		verses = append(verses, Verse{Number: verse, Text: strings.Join(fragments, " ")})
	}
	return verses, nil
}

func parseVerseID(id string) (int, int, int, bool) {
	if !strings.HasPrefix(id, "v") {
		return 0, 0, 0, false
	}
	parts := strings.Split(strings.TrimPrefix(id, "v"), "-")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}

	bookID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	chapter, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}
	verse, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, false
	}

	return bookID, chapter, verse, true
}

func verseText(n *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" && (hasClass(node, "fn") || hasClass(node, "b")) {
			return
		}
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return builder.String()
}

func formatVerses(verses []Verse, multiline bool) string {
	parts := make([]string, 0, len(verses))
	for _, verse := range verses {
		parts = append(parts, verse.Text)
	}
	if multiline {
		return strings.Join(parts, "\n")
	}
	return strings.Join(parts, " ")
}

func attr(n *html.Node, name string) string {
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, field := range strings.Fields(attr(n, "class")) {
		if field == class {
			return true
		}
	}
	return false
}

func normalizeSpace(input string) string {
	replacer := strings.NewReplacer(
		"\u00a0", " ",
		"\u202f", " ",
		"\u2007", " ",
		"\t", " ",
		"\n", " ",
		"\r", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(input)), " ")
}

func buildBookAliases() map[string]book {
	aliases := make(map[string]book)
	for _, b := range books {
		registerBookAlias(aliases, b.Name, b)
		for _, alias := range b.Aliases {
			registerBookAlias(aliases, alias, b)
		}
	}
	return aliases
}

func registerBookAlias(aliases map[string]book, alias string, b book) {
	key := normalizeBookKey(alias)
	if key == "" {
		return
	}
	aliases[key] = b
	aliases[strings.ReplaceAll(key, " ", "")] = b
}

func normalizeBookKey(input string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ä", "a", "â", "a", "Á", "a", "À", "a", "Ä", "a", "Â", "a",
		"é", "e", "è", "e", "ë", "e", "ê", "e", "É", "e", "È", "e", "Ë", "e", "Ê", "e",
		"í", "i", "ì", "i", "ï", "i", "î", "i", "Í", "i", "Ì", "i", "Ï", "i", "Î", "i",
		"ó", "o", "ò", "o", "ö", "o", "ô", "o", "Ó", "o", "Ò", "o", "Ö", "o", "Ô", "o",
		"ú", "u", "ù", "u", "ü", "u", "û", "u", "Ú", "u", "Ù", "u", "Ü", "u", "Û", "u",
		"ñ", "n", "Ñ", "n",
		".", " ",
		"\u00a0", " ",
		"\u202f", " ",
	)
	normalized := strings.ToLower(replacer.Replace(input))
	return strings.Join(strings.Fields(normalized), " ")
}
