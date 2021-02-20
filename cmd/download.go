/*
Package cmd ...
Copyright © 2020 Santeri Kääriäinen <santeri.kaariainen@iki.fi>

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
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download forecast GRIB files from FMI",
	Long: `Download weather forecasts from Finnish Meteorology Institute in GRIB format
	TODO -- something here
`,
	Run: func(cmd *cobra.Command, args []string) {
		year, _ := cmd.Flags().GetInt("year")
		month, _ := cmd.Flags().GetInt("month")
		day, _ := cmd.Flags().GetInt("day")
		hour, _ := cmd.Flags().GetInt("hour")
		dsURL := constructGribURL(year, month, day, hour)

		fileLocation, _ := cmd.Flags().GetString("output")
		downloadDataset(dsURL, fileLocation)
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	viper.SetConfigFile("fmi_config.yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	downloadCmd.Flags().Int("year", time.Now().Year(), "Year of dataset, defaults to current year.")
	downloadCmd.Flags().Int("month", int(time.Now().Month()), "Year of dataset, defaults to current month.")
	downloadCmd.Flags().Int("day", time.Now().Day(), "Day of dataset, defaults to current day.")
	downloadCmd.Flags().Int("hour", time.Now().Hour(), "Hour of dataset, defaults to current hour.")
	downloadCmd.Flags().String("output", "dataset.grb2", "Dataset output location")
}

func downloadFile(uri string) ([]byte, error) {
	res, err := http.Get(uri)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	return d, err
}

// Node ...
// Node in XML-downloaded from WFS api of FMI
type Node struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:"-"`
	Content []byte     `xml:",innerxml"`
	Nodes   []Node     `xml:",any"`
}

// UnmarshalXML ...
// Decode xml file
func (n *Node) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Attrs = start.Attr
	type node Node

	return d.DecodeElement((*node)(n), &start)
}

func walk(nodes []Node, f func(Node) bool) {
	for _, n := range nodes {
		if f(n) {
			walk(n.Nodes, f)
		}
	}
}

func getOriginTimeFromURL(url string) (time.Time, error) {
	afterOrigintime := strings.SplitAfter(url, "origintime=")[1]
	origintime := strings.Split(afterOrigintime, "&")[0]
	return time.Parse(time.RFC3339, origintime)
}

func downloadDataset(url string, outputLocation string) {
	buf, err := downloadFile(url)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(outputLocation, buf, 0644)
	if err != nil {
		panic(err)
	}
}

func constructGribURL(year int, month int, day int, hour int) string {
	ts := time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.Now().Location())
	formattedTs := ts.Format("2006-01-02T15:00:00Z")

	wfsXML, _ := downloadFile(viper.GetString("forecast.baseurl") +
		viper.GetString("forecast.query") +
		viper.GetString("forecast.parameters") +
		fmt.Sprintf("&starttime=%s&endtime=%s", formattedTs, formattedTs))

	// looping through xml from https://stackoverflow.com/questions/30256729/how-to-traverse-through-xml-data-in-golang
	buf := bytes.NewBuffer(wfsXML)
	dec := xml.NewDecoder(buf)

	var n Node
	err := dec.Decode(&n)
	if err != nil {
		panic(err)
	}

	latestGribURL := ""
	var latestOriginTime time.Time
	walk([]Node{n}, func(n Node) bool {
		if n.XMLName.Local == "fileReference" {
			contentString := string(n.Content)
			originTime, _ := getOriginTimeFromURL(contentString)
			if originTime.After(latestOriginTime) {
				latestGribURL = contentString
			}
		}
		return true
	})
	return strings.ReplaceAll(latestGribURL, "amp;", "")

}
