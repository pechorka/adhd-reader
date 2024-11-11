package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"slices"

	"github.com/pechorka/adhd-reader/pkg/fileparser/pdf"
	"github.com/pechorka/gostdlib/pkg/errs"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err.Error())
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errs.New("missing path to pdf")
	}

	filePath := os.Args[1]
	pdfBytes, err := os.ReadFile(filePath)
	if err != nil {
		return errs.Wrap(err, "failed to read pdf")
	}

	pdfText, err := pdf.PlaintText(pdfBytes)
	if err != nil {
		return errs.Wrap(err, "failed to read text from pdf")
	}

	freqs := countSentenceFreqs(pdfText)

	fileName := path.Base(filePath)
	f, err := os.Create(fileName + "freqs.csv")
	if err != nil {
		return errs.Wrap(err, "failed to open result file")
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write([]string{"sentence", "freq"})
	for _, freq := range freqs {
		w.Write([]string{freq.sentence, strconv.Itoa(freq.freq)})
	}

	return nil
}

type freq struct {
	sentence string
	freq     int
}

func countSentenceFreqs(text string) []freq {
	var stext []string
	for _, word := range strings.Split(text, " ") {
		word = strings.TrimSpace(word)
		word = strings.Trim(word, ".,!?-«»‒")
		word = strings.ToLower(word)
		if word != "" && !skip[word] {
			stext = append(stext, word)
		}
	}
	ftext := strings.Join(stext, " ")
	fmt.Printf("%d words in text\n", len(stext))

	var result []freq
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wordCount := 2
	for wordCount < 10 {
		wg.Add(1)
		go func(wc int) {
			defer wg.Done()
			defer fmt.Printf("finished analyzing %d word sentences\n", wc)
			dedup := make(map[string]bool)
			for i := 0; i < len(stext)-wc; i++ {
				sentence := strings.Join(stext[i:i+wc], " ")
				if dedup[sentence] || skip[sentence] {
					continue
				}
				dedup[sentence] = true
				f := strings.Count(ftext, sentence)
				if f > 1 {
					mu.Lock()
					result = append(result, freq{
						sentence: sentence,
						freq:     f,
					})
					if len(result)%1000 == 0 {
						fmt.Printf("found %d freqs\n", len(result))
					}
					mu.Unlock()
				}
			}
		}(wordCount)
		wordCount++
	}

	wg.Wait()
	slices.SortFunc(result, func(f1, f2 freq) int {
		return f2.freq - f1.freq
	})

	return result
}

var skip = map[string]bool{
	"—":                  true,
	"т":                  true,
	"р":                  true,
	"e":                  true,
	"п":                  true,
	"н":                  true,
	"д":                  true,
	"м":                  true,
	"л":                  true,
	"и":                  true,
	"а":                  true,
	"но":                 true,
	"или":                true,
	"да":                 true,
	"зато":               true,
	"чтобы":              true,
	"как":                true,
	"тоже":               true,
	"также":              true,
	"ни":                 true,
	"либо":               true,
	"если":               true,
	"хотя":               true,
	"потому что":         true,
	"когда":              true,
	"то":                 true,
	"раз":                true,
	"так":                true,
	"даже":               true,
	"пусть":              true,
	"но и":               true,
	"не только":          true,
	"а также":            true,
	"в то время как":     true,
	"чем":                true,
	"для того чтобы":     true,
	"чем больше":         true,
	"вместо того чтобы":  true,
	"по мере того как":   true,
	"так как":            true,
	"так что":            true,
	"как будто":          true,
	"дабы":               true,
	"едва":               true,
	"прежде чем":         true,
	"ежели":              true,
	"после того как":     true,
	"пока":               true,
	"лишь":               true,
	"пускай":             true,
	"да и":               true,
	"вроде":              true,
	"точно":              true,
	"словно":             true,
	"впрочем":            true,
	"не столько":         true,
	"иначе":              true,
	"благодаря тому что": true,
	"невзирая на то что": true,
	"нежели":             true,
	"несмотря на то что": true,
	"притом":             true,
	"при этом":           true,
	"только":             true,
	"лишь бы":            true,
	"хотя бы":            true,
	"не говоря уже о том что": true,
	"так чтобы":               true,
	"будто":                   true,
	"именно":                  true,
	"только лишь":             true,
	"не то чтобы":             true,
	"вместо того":             true,
	"несмотря":                true,
	"вследствие того что":     true,
	"с тем чтобы":             true,
	"ли":                      true,
	"бы":                      true,
	"же":                      true,
	"вот":                     true,
	"вон":                     true,
	"уж":                      true,
	"ведь":                    true,
	"неужели":                 true,
	"разве":                   true,
	"все-таки":                true,
	"всего лишь":              true,
	"едва ли":                 true,
	"все же":                  true,
	"исключительно":           true,
	"почти":                   true,
	"будто бы":                true,
	"как раз":                 true,
	"вроде бы":                true,
	"чуть ли":                 true,
	"просто":                  true,
	"никак":                   true,
	"неужто":                  true,
	"все же таки":             true,
	"да уж":                   true,
	"наверно":                 true,
	"поистине":                true,
	"тем не менее":            true,
	"в точности":              true,
	"было бы":                 true,
	"всего лишь только":       true,
	"едва ли не":              true,
	"на самом деле":           true,
	"в точности до":           true,
	"ах":                      true,
	"ох":                      true,
	"ой":                      true,
	"эх":                      true,
	"ну":                      true,
	"эй":                      true,
	"ау":                      true,
	"фу":                      true,
	"ух":                      true,
	"ой-ой":                   true,
	"ай":                      true,
	"вау":                     true,
	"ах ты":                   true,
	"ух ты":                   true,
	"ой-ой-ой":                true,
	"ну-ну":                   true,
	"ага":                     true,
	"фи":                      true,
	"брр":                     true,
	"вот те на":               true,
	"тсс":                     true,
	"уф":                      true,
	"хм":                      true,
	"пожалуйста":              true,
	"чур":                     true,
	"постой":                  true,
	"тихо":                    true,
	"ей-богу":                 true,
	"бис":                     true,
	"внимание":                true,
	"господи":                 true,
	"браво":                   true,
	"алло":                    true,
	"ура":                     true,
	"ба":                      true,
	"цыц":                     true,
	"эй-эй":                   true,
	"эх-эх":                   true,
	"кыш":                     true,
	"прощай":                  true,
	"тьфу":                    true,
	"ахах":                    true,
	"ой-вей":                  true,
	"охох":                    true,
	"упс":                     true,
	"ля-ля":                   true,
	"в":                       true,
	"на":                      true,
	"за":                      true,
	"по":                      true,
	"под":                     true,
	"из":                      true,
	"к":                       true,
	"с":                       true,
	"у":                       true,
	"о":                       true,
	"от":                      true,
	"до":                      true,
	"для":                     true,
	"при":                     true,
	"про":                     true,
	"через":                   true,
	"об":                      true,
	"над":                     true,
	"перед":                   true,
	"без":                     true,
	"между":                   true,
	"вокруг":                  true,
	"ради":                    true,
	"вне":                     true,
	"после":                   true,
	"около":                   true,
	"вместо":                  true,
	"из-за":                   true,
	"из-под":                  true,
	"среди":                   true,
	"благодаря":               true,
	"вследствие":              true,
	"навстречу":               true,
	"насчет":                  true,
	"посредством":             true,
	"внутри":                  true,
	"вдоль":                   true,
	"помимо":                  true,
	"сверх":                   true,
	"по поводу":               true,
	"напротив":                true,
	"вблизи":                  true,
	"вдали":                   true,
	"по причине":              true,
	"согласно":                true,
	"сообразно":               true,
	"несмотря на":             true,
	"в зависимости от":        true,
	"in":                      true,
	"on":                      true,
	"at":                      true,
	"by":                      true,
	"with":                    true,
	"about":                   true,
	"against":                 true,
	"between":                 true,
	"into":                    true,
	"through":                 true,
	"during":                  true,
	"before":                  true,
	"after":                   true,
	"above":                   true,
	"below":                   true,
	"to":                      true,
	"from":                    true,
	"up":                      true,
	"down":                    true,
	"over":                    true,
	"under":                   true,
	"again":                   true,
	"out":                     true,
	"around":                  true,
	"near":                    true,
	"behind":                  true,
	"along":                   true,
	"following":               true,
	"across":                  true,
	"beside":                  true,
	"besides":                 true,
	"except":                  true,
	"towards":                 true,
	"upon":                    true,
	"within":                  true,
	"without":                 true,
	"alongside":               true,
	"amid":                    true,
	"among":                   true,
	"concerning":              true,
	"despite":                 true,
	"inside":                  true,
	"outside":                 true,
	"regarding":               true,
	"throughout":              true,
	"toward":                  true,
	"via":                     true,
	"per":                     true,
	"oh":                      true,
	"ah":                      true,
	"wow":                     true,
	"ouch":                    true,
	"oops":                    true,
	"hey":                     true,
	"alas":                    true,
	"bravo":                   true,
	"yay":                     true,
	"yikes":                   true,
	"hmm":                     true,
	"uh-oh":                   true,
	"aha":                     true,
	"hooray":                  true,
	"shh":                     true,
	"whoa":                    true,
	"oopsie":                  true,
	"ugh":                     true,
	"eek":                     true,
	"phew":                    true,
	"tsk-tsk":                 true,
	"yippee":                  true,
	"boo":                     true,
	"jeez":                    true,
	"ahem":                    true,
	"ew":                      true,
	"golly":                   true,
	"oops-a-daisy":            true,
	"blimey":                  true,
	"bah":                     true,
	"meh":                     true,
	"whoops":                  true,
	"bingo":                   true,
	"wowza":                   true,
	"huh":                     true,
	"ho-ho":                   true,
	"hurray":                  true,
	"yum":                     true,
	"zoinks":                  true,
	"huzzah":                  true,
	"aha!":                    true,
	"gee":                     true,
	"and":                     true,
	"but":                     true,
	"or":                      true,
	"nor":                     true,
	"for":                     true,
	"yet":                     true,
	"so":                      true,
	"because":                 true,
	"since":                   true,
	"if":                      true,
	"though":                  true,
	"although":                true,
	"unless":                  true,
	"while":                   true,
	"whereas":                 true,
	"whether":                 true,
	"as":                      true,
	"than":                    true,
	"once":                    true,
	"until":                   true,
	"when":                    true,
	"whenever":                true,
	"where":                   true,
	"wherever":                true,
	"even though":             true,
	"even if":                 true,
	"provided that":           true,
	"in case":                 true,
	"in order that":           true,
	"as soon as":              true,
	"as long as":              true,
	"not":                     true,
	"just":                    true,
	"only":                    true,
	"even":                    true,
	"too":                     true,
	"very":                    true,
	"quite":                   true,
	"almost":                  true,
	"much":                    true,
	"rather":                  true,
	"still":                   true,
	"merely":                  true,
	"hardly":                  true,
	"scarcely":                true,
	"barely":                  true,
	"exactly":                 true,
	"simply":                  true,
	"truly":                   true,
	"really":                  true,
	"certainly":               true,
	"definitely":              true,
	"just about":              true,
	"no":                      true,
	"yes":                     true,
	"indeed":                  true,
	"possibly":                true,
	"probably":                true,
	"maybe":                   true,
	"perhaps":                 true,
	"surely":                  true,
	"can":                     true,
	"could":                   true,
	"may":                     true,
	"might":                   true,
	"must":                    true,
	"shall":                   true,
	"should":                  true,
	"will":                    true,
	"would":                   true,
	"ought":                   true,
	"need":                    true,
	"dare":                    true,
	"a":                       true,
	"an":                      true,
	"the":                     true,
}