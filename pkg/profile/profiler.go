package profile

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"ngoperf/pkg/myhttp"

	"github.com/cheggaaa/pb/v3"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type profileResult struct {
	requestResponseTime []int64
	responseSize        []int64
	fatalError          map[string]int
	status              map[string]int
	statusCode          map[int]int
	responseBody        string
}

// Profiler is used to get of profile a url depending on its setting
type Profiler struct {
	numRequest int
	numWorker  int
	http10     bool
	verbose    bool
	isGetter   bool
}

// NewProfiler returns a new Profiler
// Profiler request numRequest times with numWorker and prints profile summary
func NewProfiler(numProfile int, numWorker int, verbose, http10 bool) (p *Profiler) {
	p = &Profiler{
		numRequest: numProfile,
		numWorker:  numWorker,
		http10:     http10,
		verbose:    verbose,
		isGetter:   false,
	}
	return p
}

// NewGetter returns a new Profiler with the Getter setting
// Getter prints response body (and response header if verbose is set)
func NewGetter(http10 bool, verbose bool) (p *Profiler) {
	p = &Profiler{
		numRequest: 1,
		numWorker:  1,
		http10:     http10,
		verbose:    verbose,
		isGetter:   true,
	}
	return p
}

// RunProfile profiles the url
func (p *Profiler) RunProfile(url string) {
	result := &profileResult{
		status:     make(map[string]int),
		fatalError: make(map[string]int),
		statusCode: make(map[int]int),
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	records := make(chan *myhttp.Response, p.numRequest)
	jobs := make(chan int, p.numRequest)
	var bar *pb.ProgressBar
	if !p.isGetter {
		bar = pb.StartNew(p.numRequest)
	}

	var counter chan int
	if bar != nil {
		counter = make(chan int)
		go func() {
			for range counter {
				bar.Increment()
			}
		}()
	}

	var wg sync.WaitGroup
	for i := 0; i < p.numWorker; i++ {
		wg.Add(1)
		cfg := &workerCFG{http10: p.http10, verbose: p.verbose, useCounter: bar != nil}
		go worker(&wg, counter, jobs, records, url, cfg)
	}

	go func() {
		for i := 0; i < p.numRequest; i++ {
			jobs <- i
		}
		close(jobs)
	}()
	wg.Wait()
	close(records)
	if counter != nil {
		close(counter)
	}
	if bar != nil {
		bar.Finish()
	}
	aggregateResult(p, records, result)
	if p.isGetter {
		fmt.Println(result.responseBody)
	} else {
		printProfileResults(result)
	}
	if len(result.fatalError) > 0 {
		printErrors(result)
	}
}

type workerCFG struct {
	http10     bool
	verbose    bool
	useCounter bool
}

func worker(wg *sync.WaitGroup, counter chan int, jobs chan int, records chan *myhttp.Response, url string, cfg *workerCFG) {
	defer wg.Done()
	client := &myhttp.Client{Verbose: cfg.verbose, HTTP10: cfg.http10}
	defer func() {
		if client.Conn != nil {
			client.Conn.Close()
		}
	}()
	for range jobs {
		rc, err := client.GET(url)
		if err != nil {
			if cfg.verbose {
				errStr := fmt.Sprintf("Send GET rerror %s: %s", url, err.Error())
				fmt.Println(errStr)
			}
			rc = &myhttp.Response{Status: err.Error()}
		}
		if cfg.useCounter {
			counter <- 1
		}
		records <- rc
	}
}

func aggregateResult(p *Profiler, records chan *myhttp.Response, result *profileResult) {
	for rec := range records {
		if rec.StatusCode == 0 {
			result.fatalError[rec.Status]++
			continue
		}
		result.status[rec.Status]++
		if p.isGetter {
			result.responseBody = rec.ResponseBody
			continue
		}
		result.requestResponseTime = append(result.requestResponseTime, rec.ResponseTime)
		result.responseSize = append(result.responseSize, rec.ResponseSize)
		result.statusCode[rec.StatusCode]++
	}
}

func printProfileResults(result *profileResult) {
	n := len(result.responseSize)
	if n <= 0 {
		fmt.Println("No Result.")
		return
	}
	fmt.Println()
	printSuccessRate(n, &result.statusCode)
	printStatusSummary(result.status)
	printSpeedSummary(result.requestResponseTime)
	printSizeSummary(result.responseSize)
}

func printSuccessRate(n int, statusCode *map[int]int) {
	success := 0
	for st, cnt := range *statusCode {

		if st/100 == 2 {
			success += cnt
		}
	}
	fmt.Println("The number of requests: " + strconv.Itoa(n))
	fmt.Println(fmt.Sprintf("The success rate is: %.1f %%", float32(success)*100/float32(n)))
}

func printStatusSummary(status map[string]int) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"status", "count"})
	summary := [][]string{}
	for st, cnt := range status {
		data := []string{st, prettyInt(cnt)}
		if !strings.HasPrefix(st, "2") {
			table.Rich(data, []tablewriter.Colors{{tablewriter.BgRedColor}})
		} else {
			table.Rich(data, []tablewriter.Colors{{tablewriter.BgGreenColor}})
		}
	}
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgBlackColor},
	)
	table.AppendBulk(summary)
	table.Render()
}

func printSpeedSummary(intervals []int64) {
	fmt.Println("\nThe Summary of Response Time (ms):")
	sort.Slice(intervals, func(i, j int) bool { return intervals[i] < intervals[j] })
	n := len(intervals)
	var sum int64 = 0
	for _, val := range intervals {
		sum += val
	}
	fast := prettyInt64(intervals[0])
	slow := prettyInt64(intervals[n-1])
	mean := prettyInt64(sum / int64(n))
	median := prettyInt64(intervals[0+n/2])

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"fast", "slow", "mean", "median"})
	summary := [][]string{
		{fast, slow, mean, median},
	}
	table.AppendBulk(summary)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgBlackColor})
	table.Render()
}

func printSizeSummary(responseSize []int64) {
	fmt.Println("\nThe Smallest and Largest Responses (bytes):")
	sort.Slice(responseSize, func(i, j int) bool { return responseSize[i] < responseSize[j] })
	n := len(responseSize)
	smallest := prettyInt64(responseSize[0])
	largest := prettyInt64(responseSize[n-1])

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"smallest", "largest"})
	summary := [][]string{
		{smallest, largest},
	}
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgBlackColor},
	)
	table.AppendBulk(summary)
	table.Render()
}

var printer = message.NewPrinter(language.English)

func prettyInt64(val int64) string {
	return printer.Sprintf("%d", val)
}

func prettyInt(val int) string {
	return printer.Sprintf("%d", val)
}

func printErrors(result *profileResult) {
	fmt.Println("\nFatal Errors:")
	printStatusSummary(result.fatalError)
}
