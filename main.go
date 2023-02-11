package main

import (
	"flag"
	"fmt"
	ecsService "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"strings"
)

var (
	accessKeyId     = flag.String("accessKeyId", "", "Your accessKeyId of cloud account")
	accessKeySecret = flag.String("accessKeySecret", "", "Your accessKeySecret of cloud account")
	region          = flag.String("region", "cn-hangzhou", "The region of spot instances")
	cpu             = flag.Int("mincpu", 1, "Min cores of spot instances")
	memory          = flag.Int("minmem", 2, "Min memory of spot instances")
	maxCpu          = flag.Int("maxcpu", 32, "Max cores of spot instances ")
	maxMemory       = flag.Int("maxmem", 64, "Max memory of spot instances")
	family          = flag.String("family", "", "The spot instance family you want (e.g. ecs.n1,ecs.n2)")
	cutoff          = flag.Int("cutoff", 2, "Discount of the spot instance prices")
	limit           = flag.Int("limit", 20, "Limit of the spot instances")
	resolution      = flag.Int("resolution", 7, "The window of price history analysis")
	regions         = flag.String("regions", "", "The regions of spot instances, * / cn / cn,ap")
)

func main() {
	flag.Parse()

	var regionList []string

	if *regions != "" {
		// 必须设置一个 regionId，否则无法正常调用
		c, e := ecsService.NewClientWithAccessKey("cn-hangzhou", *accessKeyId, *accessKeySecret)
		if e != nil {
			panic(fmt.Sprintf("Failed to create ecs client,because of %v", e))
		}
		request := ecsService.CreateDescribeRegionsRequest()
		response, err := c.DescribeRegions(request)
		if e != nil {
			panic(fmt.Sprintf("Failed to describe regions,because of %v", err))
		}
		regionsFilter := strings.Split(*regions, ",")
		for _, r := range response.Regions.Region {
			if *regions == "*" {
				regionList = append(regionList, r.RegionId)
			} else {
				for _, prefix := range regionsFilter {
					if strings.HasPrefix(r.RegionId, prefix) {
						regionList = append(regionList, r.RegionId)
					}
				}
			}
		}
		fmt.Printf("Available regions: %s\n", strings.Join(regionList, ","))
	} else {
		regionList = []string{*region}
	}

	allPrices := SortedInstancePrices{}
	for _, r := range regionList {
		fmt.Printf("Processing region: %s\n", r)

		client, err := ecsService.NewClientWithAccessKey(r, *accessKeyId, *accessKeySecret)
		if err != nil {
			panic(fmt.Sprintf("Failed to create ecs client,because of %v", err))
		}

		metastore := NewMetaStore(client)

		metastore.Initialize(r)

		instanceTypes := metastore.FilterInstances(*cpu, *memory, *maxCpu, *maxMemory, *family)

		historyPrices := metastore.FetchSpotPrices(instanceTypes, *resolution)

		sortedInstancePrices := metastore.SpotPricesAnalysis(historyPrices)

		allPrices = append(allPrices, sortedInstancePrices...)
	}

	metastore := NewMetaStore(nil)
	metastore.PrintPriceRank(allPrices, *cutoff, *limit)
}
