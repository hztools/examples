// {{{ Copyright (c) Paul R. Tagliamonte <paul@k3xec.com>, 2023
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE. }}}

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"hz.tools/cli"
	"hz.tools/fftw"
	"hz.tools/fm"
	"hz.tools/pulseaudio"
	"hz.tools/rf"
	"hz.tools/sdr"
	"hz.tools/sdr/stream"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fm",
	Short: "listen to fm radio",
	Long:  `Tune to an analog FM radio station`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, _, _, err := cli.LoadSDR(cmd)
		if err != nil {
			return err
		}
		defer dev.Close()

		sinkName, err := cmd.Flags().GetString("sink-name")
		if err != nil {
			return err
		}

		bandwidthStr, err := cmd.Flags().GetString("bandwidth")
		if err != nil {
			return err
		}

		gain, err := cmd.Flags().GetFloat32("gain")
		if err != nil {
			return err
		}

		downsample, err := cmd.Flags().GetUint("downsample")
		if err != nil {
			return err
		}

		switch bandwidthStr {
		case "broadcast":
			bandwidthStr = "150KHz"
		case "narrowband":
			bandwidthStr = "5KHz"
		}

		bandwidth, err := rf.ParseHz(bandwidthStr)
		if err != nil {
			return err
		}

		rcv := dev.(sdr.Receiver)
		readCloser, err := rcv.StartRx()
		if err != nil {
			return err
		}
		defer readCloser.Close()
		var reader sdr.Reader = readCloser

		reader, err = stream.ConvertReader(reader, sdr.SampleFormatC64)
		if err != nil {
			return err
		}

		demod, err := fm.Demodulate(reader, fm.DemodulatorConfig{
			CenterFrequency: rf.Hz(0),
			Deviation:       bandwidth / 2,
			Downsample:      downsample,
			Planner:         fftw.Plan,
		})
		if err != nil {
			return err
		}

		speaker, err := pulseaudio.NewWriter(pulseaudio.Config{
			Format:     pulseaudio.SampleFormatFloat32NE,
			Rate:       demod.SampleRate(),
			AppName:    "rf",
			StreamName: "fm",
			Channels:   1,
			SinkName:   sinkName,
		})
		if err != nil {
			return err
		}

		buf := make([]float32, 1024*64)
		for {
			i, err := demod.Read(buf)
			if err != nil {
				return err
			}
			if i == 0 {
				panic("zero read")
			}
			for j := 0; j < i; j++ {
				buf[j] *= gain
			}
			if err := speaker.Write(buf[:i]); err != nil {
				return err
			}
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	flags := rootCmd.Flags()

	flags.String("bandwidth", "broadcast", "Bandwidth for the fm signal [broadcast|narrowband|<hz>]")
	flags.Uint("downsample", 8, "Samples to downsample for audio playback")
	flags.String("sink-name", "", "pulseaudio sink name")
	flags.Float32("gain", 0.75, "amount of gain on the signal")

	cli.RegisterSDRFlags(rootCmd)
}

// vim: foldmethod=marker
