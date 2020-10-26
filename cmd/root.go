package cmd

import (
	"fmt"
	"ngoperf/pkg/profile"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	reqURL     string
	numProfile int
	numWorker  int
	http10     bool
	verbose    bool
	sleepTime  int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ngoperf [-u url] [-0] ...",
	Short: "ngoperf is a Go implemented CLI tool for profiling websites",
	Long:  `ngoperf is a Go implemented CLI tool for profiling websites`,
	Example: `ngoperf get -u https://hi.wanghy917.workers.dev/links
ngoperf profile --url=stackoverflow.com -p=1000 -w=100
ngoperf profile --url=www.cloudflare.com:443 -p=1000 -w=100`,
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Send mutilple HTTP GET requests to a url, and output summary about status, time and size",
	Long: `Send mutilple HTTP GET requests to a url, and output summary about status, time and size
The number of request and number of workers to send request could be set with -u and -v options, see below for details`,

	Run: func(cmd *cobra.Command, args []string) {
		profiler := profile.NewProfiler(numProfile, numWorker, verbose, http10, sleepTime)
		profiler.RunProfile(reqURL)
	},
	Example: "ngoperf profile -u=www.google.com -p=2000 -w=400",
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Send HTTP GET to a url and print the response",
	Long: `Send HTTP GET to a url and print the response
The get command print HTTP response body only by default. To print request and response header, add the -v option.`,
	Run: func(cmd *cobra.Command, args []string) {
		profiler := profile.NewGetter(http10, verbose)
		profiler.RunProfile(reqURL)
	},
	Example: "ngoperf get -vz -u http://hi.wanghy917.workers.dev/links",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	profileCmd.Flags().BoolVarP(&http10, "http10", "z", false, "use HTTP/1.0 to request\nnhoprtg use HTTP/1.1 by default")
	profileCmd.Flags().StringVarP(&reqURL, "url", "u", "", "request url\nngoperf use https with port 443 to connect if protocol and port are not included")
	profileCmd.MarkFlagRequired("url")
	profileCmd.Flags().IntVarP(&numProfile, "np", "p", 100, "num of request")
	profileCmd.Flags().IntVarP(&numWorker, "nw", "w", 5, "num of worker")
	profileCmd.Flags().IntVarP(&sleepTime, "sleep", "s", 0, "sleep time between requests\nngoperf randomly sleep 0 to s seconds between the requests")
	rootCmd.AddCommand(profileCmd)

	getCmd.Flags().BoolVarP(&http10, "http10", "z", false, "use HTTP/1.0 to request\nnhoprtg use HTTP/1.1 by default")
	getCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print request and response header")
	getCmd.Flags().StringVarP(&reqURL, "url", "u", "", "request url\nngoperf use https with port 443 to connect if protocol and port are not included")
	getCmd.MarkFlagRequired("url")

	rootCmd.AddCommand(getCmd)
}
