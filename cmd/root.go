/*
Copyright © 2020 Han-Yi Wang wanghy917@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ngoperf [-u url] [-0] ...",
	Short: "ngoperf is a Go implemented CLI tool for profiling websites",
	Long:  `ngoperf is a Go implemented CLI tool for profiling websites`,
	Example: `ngoperf get -u https://hi.wanghy917.workers.dev/links
ngoperf profile --url=stackoverflow.com -p=1000 -w=100`,
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Use HTTP GET to profiling the url",
	Long:  `Use HTTP GET to profiling the url`,
	Run: func(cmd *cobra.Command, args []string) {
		profiler := profile.NewProfiler(numProfile, numWorker, verbose, http10)
		profiler.RunProfile(reqURL)
	},
	Example: "ngoperf profile -u=stackoverflow.com -p=1000 -w=100",
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Use HTTP GET to request the url and print the response",
	Long:  `Use HTTP GET to request the url and print the response`,
	Run: func(cmd *cobra.Command, args []string) {
		profiler := profile.NewGetter(http10, verbose)
		profiler.RunProfile(reqURL)
	},
	Example: "ngoperf get -u=https://cloudflare-worker-practice.wanghy917.workers.dev/links",
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
	rootCmd.PersistentFlags().BoolVarP(&http10, "http10", "z", false, "use HTTP/1.0 to request instead of 1.1")
	rootCmd.PersistentFlags().StringVarP(&reqURL, "url", "u", "", "request url")
	rootCmd.MarkPersistentFlagRequired("url")
	profileCmd.Flags().IntVarP(&numProfile, "np", "p", 100, "num of request")
	profileCmd.Flags().IntVarP(&numWorker, "nw", "w", 20, "num of worker")
	rootCmd.AddCommand(profileCmd)

	getCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print request and response header")
	rootCmd.AddCommand(getCmd)
}
