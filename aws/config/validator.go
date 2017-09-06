package awsconfig

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/chzyer/readline"
)

func ParseRegion(i string) (interface{}, error) {
	if !IsValidRegion(i) {
		return i, fmt.Errorf("'%s' is not a valid region", i)
	}
	return i, nil
}

func WarningChangeRegion(i interface{}) {
	if firstInstall, err := strconv.ParseBool(os.Getenv("__AWLESS_FIRST_INSTALL")); !firstInstall || err != nil {
		fmt.Fprint(os.Stderr, "You might want to update your default AMI with `awless config set instance.image $(awless search images amazonlinux --id-only --silent)`\n")
	}
}

func ParseInstanceType(i string) (interface{}, error) {
	if !isValidInstanceType(i) {
		return i, fmt.Errorf("'%s' is not a valid instance type", i)
	}
	return i, nil
}

func StdinRegionSelector() string {
	var regionItems []readline.PrefixCompleterInterface
	for _, r := range allRegions() {
		regionItems = append(regionItems, readline.PcItem(r))
	}
	var regionCompleter = readline.NewPrefixCompleter(regionItems...)

	fmt.Println("Please enter one region: (Ctrl+C to quit, Tab for completion)")
	var region string
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "> ",
		AutoComplete: regionCompleter,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while selecting region: %s", err)
		return ""
	}
	defer rl.Close()

	for !IsValidRegion(region) {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "error while selecting region: %s", err)
			return ""
		}

		region = strings.TrimSpace(line)
		if !IsValidRegion(region) {
			fmt.Fprintf(os.Stderr, "'%s' is not a valid region\n", region)
		}
	}

	return region
}

func StdinInstanceTypeSelector() string {
	fmt.Println("Please choose one instance type")
	fmt.Println()
	fmt.Println("Here are few examples:")

	var instanceType string
	t := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(t, "\tinstance type\tvCPU\tMemory (GiB)")
	fmt.Fprintln(t, "\tt2.nano\t1\t0.5")
	fmt.Fprintln(t, "\tt2.micro\t1\t1")
	fmt.Fprintln(t, "\tt2.small\t1\t2")
	fmt.Fprintln(t, "\tt2.medium\t2\t4")
	fmt.Fprintln(t, "\tt2.large\t2\t8")
	fmt.Fprintln(t, "\tt2.xlarge\t4\t16")
	fmt.Fprintln(t, "\tt2.2xlarge\t8\t32")
	fmt.Fprintln(t, "\tm4.large\t2\t8")
	fmt.Fprintln(t, "\tm4.xlarge\t4\t16")
	fmt.Fprintln(t, "\tc4.large\t2\t3.75")
	fmt.Fprintln(t, "\tc4.xlarge\t4\t7.5")
	fmt.Fprintln(t, "\t...")
	t.Flush()

	fmt.Println()
	fmt.Print("Value ? > ")
	fmt.Scan(&instanceType)
	for !isValidInstanceType(instanceType) {
		fmt.Printf("'%s' is not a valid instance type\n", instanceType)
		fmt.Print("Value ? > ")
		fmt.Scan(&instanceType)
	}
	return instanceType
}

func IsValidRegion(given string) bool {
	reg, _ := regexp.Compile("^(us|eu|ap|sa|ca)\\-\\w+\\-\\d+$")
	regChina, _ := regexp.Compile("^cn\\-\\w+\\-\\d+$")
	regUsGov, _ := regexp.Compile("^us\\-gov\\-\\w+\\-\\d+$")

	return reg.MatchString(given) || regChina.MatchString(given) || regUsGov.MatchString(given)
}

func isValidInstanceType(given string) bool {
	return regexp.MustCompile("\\w+\\.\\w+").MatchString(given)
}

func allRegions() []string {
	var regions sort.StringSlice
	partitions := endpoints.DefaultResolver().(endpoints.EnumPartitions).Partitions()
	for _, p := range partitions {
		for id := range p.Regions() {
			regions = append(regions, id)
		}
	}
	sort.Sort(regions)
	return regions
}
