// Copyright © 2019 Jonathan Pentecost <pentecostjonathan@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"github.com/vishen/go-chromecast/tts"
)

// ttsCmd represents the tts command
var ttsCmd = &cobra.Command{
	Use:   "tts <message>",
	Short: "text-to-speech",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 || args[0] == "" {
			fmt.Printf("expected exactly one argument to convert to speech\n %s", args)
			return
		}

		app, err := castApplication(cmd, args)
		if err != nil {
			fmt.Printf("unable to get cast application: %v\n", err)
			return
		}

		googleServiceAccount, _ := cmd.Flags().GetString("google-service-account")
		if googleServiceAccount == "" {
			fmt.Printf("--google-service-account is required\n")
			return
		}

		languageCode, _ := cmd.Flags().GetString("language-code")
		cache, _ := cmd.Flags().GetBool("cache")
		text := args[0]

		fmt.Printf("play cache=%v text=%s\n", cache, text)

		f, err := ttsFile(text, cache)
		if err != nil {
			fmt.Printf("unable to create temp file: %v\n", err)
			return
		}
		if !fileExistsAndNonZeroSize(f.Name()) {

			fmt.Println("calling TTS service")
			googleServiceAccountJson, err := ioutil.ReadFile(googleServiceAccount)
			if err != nil {
				fmt.Printf("unable to open google service account file: %v\n", err)
				return
			}

			data, err := tts.Create(text, googleServiceAccountJson, languageCode)
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
			if _, err := f.Write(data); err != nil {
				fmt.Printf("unable to write to temp file: %v\n", err)
				return
			}
			if err := f.Close(); err != nil {
				fmt.Printf("unable to close temp file: %v\n", err)
				return
			}
		}

		if !cache {
			defer os.Remove(f.Name())
		}

		if err := app.Load(f.Name(), "audio/mp3", false, false, false); err != nil {
			fmt.Printf("unable to load media to device: %v\n", err)
			return
		}
		return
	},
}

func ttsFile(text string, cache bool) (f *os.File, err error) {
	name := "go-chromecast-tts*.mp3"
	f, err = nil, nil
	if cache {
		h := sha1.New()
		h.Write([]byte(text))
		bs := h.Sum(nil)
		name = fmt.Sprintf("%s/%x.mp3", os.TempDir(), bs)
		if fileExistsAndNonZeroSize(name) {
			f, err = os.Open(name)
		} else {
			f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
		}
	} else {
		f, err = ioutil.TempFile("", name)
	}
	return f, err
}

func fileExistsAndNonZeroSize(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

func init() {
	rootCmd.AddCommand(ttsCmd)
	ttsCmd.Flags().String("google-service-account", "", "google service account JSON file")
	ttsCmd.Flags().String("language-code", "en-US", "text-to-speech Language Code (de-DE, ja-JP,...)")
	ttsCmd.Flags().Bool("cache", false, "Cache TTS results")
}
