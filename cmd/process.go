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
	"os"
	"strconv"
	"strings"

	"fmt"
	"image/png"

	"github.com/Napuu/go-gdal"
	"github.com/anthonynsimon/bild/blur"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// processCmd represents the process command
var processCmd = &cobra.Command{
	Use:   "process",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		inputLocation, _ := cmd.Flags().GetString("input")
		outputLocation, _ := cmd.Flags().GetString("output")
		blurIntensity, _ := cmd.Flags().GetInt("blur")
		// this should be done in-memory, but using existing gdal utils
		// is faster implementation wise :-)
		tmp1 := uuid.New().String() + ".tiff"
		tmp2 := uuid.New().String() + ".tiff"
		tmp3 := uuid.New().String() + ".tiff"
		tmp4 := uuid.New().String() + ".png"

		grib2tiff(inputLocation, tmp1)
		tiff2projectedTiff(tmp1, tmp2)
		projectedTiff2croppedPng(tmp2, tmp3)
		blurImg(tmp3, tmp4, blurIntensity)
		formatConversion(tmp4, outputLocation)

		os.Remove(tmp1)
		os.Remove(tmp2)
		os.Remove(tmp3)
		os.Remove(tmp3+ ".aux.xml")
		os.Remove(tmp4)
	},
}

func init() {
	rootCmd.AddCommand(processCmd)

	processCmd.Flags().String("input", "dataset.grb2", "Dataset input location")
	processCmd.Flags().String("output", "output.jpeg", "Dataset output location")
	processCmd.Flags().Int("blur", 20, "Gaussian blur intensity")

}

// BandInfo ...
// Description about individual band of grib2 file
type BandInfo struct {
	band           string
	gribDiscipline string
	gribComment    string
	gribUnit       string
}

func grib2tiff(src string, dst string) {
	dsIn, _ := gdal.Open(src, gdal.ReadOnly)
	dsOut := gdal.GDALTranslate(dst, dsIn, []string{"-CO", "COMPRESS=LZW"})
	defer dsOut.Close()
	defer dsIn.Close()
}

func blurImg(src string, dst string, blurIntensity int) {
	imagePath, _ := os.Open(src)
	defer imagePath.Close()

	srcImage, er := png.Decode(imagePath)
	if er != nil {
		fmt.Println(er)
	}

	dstImage := blur.Gaussian(srcImage, float64(blurIntensity))
	newImage, _ := os.Create(dst)
	defer newImage.Close()
	png.Encode(newImage, dstImage)
}

func formatConversion(src string, dst string) {
	dsIn, _ := gdal.Open(src, gdal.ReadOnly)
	dsOut := gdal.GDALTranslate(dst, dsIn, []string{"-CO", "COMPRESS=LZW"})
	defer dsOut.Close()
	defer dsIn.Close()
}

func tiff2projectedTiff(src string, dst string) {
	dsIn, _ := gdal.Open(src, gdal.ReadOnly)
	var nilDs gdal.Dataset
	dsOut := gdal.GDALWarp(dst, nilDs, []gdal.Dataset{dsIn}, []string{"-s_srs", "EPSG:4326", "-t_srs", "EPSG:3857", "-CO", "COMPRESS=LZW"})
	defer dsOut.Close()
	defer dsIn.Close()
}

func projectedTiff2croppedPng(src string, dst string) {
	dsIn, _ := gdal.Open(src, gdal.ReadOnly)
	// bigger bbox
	// left x -23.730469
	// left y 45.460131
	// right x 44.472656
	// right y 75.095633
	//bbox := []string{"10459", "12517984", "3845201", "7043489"}
	//bbox := []string{leftx, uppery, rightx, lowery}
	bbox := []string{"-42", "80", "48", "40"}
	dsOut := gdal.GDALTranslate(dst, dsIn, append([]string{"-of", "PNG", "-ot", "byte", "-scale_1", "-50", "50", "-scale_2", "-50", "50", "-projwin_srs", "EPSG:4326", "-projwin"}, bbox...))
	defer dsOut.Close()
	defer dsIn.Close()
}

func getGribBandInfo(gdalinfoString string) []BandInfo {
	rows := strings.Split(gdalinfoString, "\n")

	bandInfos := []BandInfo{}

	activeBand := -1
	currentBandInfo := BandInfo{}
	for _, s := range rows {
		if strings.Index(s, "Band ") == 0 {

			if activeBand != -1 {
				currentBandInfo.band = strconv.Itoa(activeBand)
				bandInfos = append(bandInfos, currentBandInfo)
			}
			newBand, err := strconv.Atoi(strings.Split(s, " ")[1])
			if err != nil {
				panic(err)
			}
			activeBand = newBand
			continue
		}

		if strings.Index(s, "GRIB_COMMENT") != -1 {
			currentBandInfo.gribComment = strings.Split(s, "=")[1]

			// Apparently wind direction is usually calculated from wind speeds u- and
			// v-components. FMI however offers that also as a variable so we're using that.
			// It's not included in grb2 standard so this "hack" is needed
			if strings.Index(currentBandInfo.gribComment, "192") != -1 {
				currentBandInfo.gribComment = "Wind direction [°]"
			}
		}
		if strings.Index(s, "GRIB_DISCIPLINE") != -1 {
			currentBandInfo.gribDiscipline = strings.Split(s, "=")[1]
		}
		if strings.Index(s, "GRIB_UNIT") != -1 {
			currentBandInfo.gribUnit = strings.Split(s, "=")[1]

			// see gribComment for explanation
			if currentBandInfo.gribUnit == "[-]" {
				currentBandInfo.gribUnit = "[°]"
			}
		}
	}
	currentBandInfo.band = strconv.Itoa(activeBand)
	bandInfos = append(bandInfos, currentBandInfo)

	return bandInfos
}
