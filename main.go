package main

import (
	"log"
	"strings"
	"time"

	"github.com/scgolang/midi"
	"github.com/scgolang/sc"
)

func main() {
	devices, err := midi.Devices()
	if err != nil {
		log.Fatal(err)
	}
	var keystation *midi.Device
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), "keystation") {
			keystation = d
			break
		}
	}
	if keystation == nil {
		log.Fatal("no keystation detected")
	}
	if err := keystation.Open(); err != nil {
		log.Fatal(err)
	}
	packets, err := keystation.Packets()
	if err != nil {
		log.Fatal(err)
	}
	client, err := sc.NewClient("udp", "127.0.0.1:0", "127.0.0.1:57120", 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	def := sc.NewSynthdef("bells", bellFunc)

	if err := client.SendDef(def); err != nil {
		log.Fatal(err)
	}
	group, err := client.AddDefaultGroup()
	if err != nil {
		log.Fatal(err)
	}
	for pkt := range packets {
		ctls := map[string]float32{
			"amp":  float32(pkt[2]) / float32(127),
			"dur":  float32(2),
			"fund": sc.Midicps(float32(pkt[1])),
		}
		id := client.NextSynthID()
		if _, err := group.Synth("bells", id, sc.AddToTail, ctls); err != nil {
			log.Fatal(err)
		}
	}
}

var bellFunc sc.UgenFunc = func(params sc.Params) sc.Ugen {
	var (
		amp  = params.Add("amp", 0.9)
		dur  = params.Add("dur", 1)
		fund = params.Add("fund", 440)
		data = normalizeSum([][4]float32{
			{0.58, 0, 1, 1},
			{0.58, 1, 0.67, 0.9},
			{0.91, 0, 1, 0.65},
			{0.91, 1.7, 1.8, 0.55},
			{1.6, 0, 1.67, 0.35},
			{1.2, 0, 2.67, 0.325},
			{2, 0, 1.46, 0.25},
			{2.7, 0, 1.33, 0.2},
			{3, 0, 1.33, 0.15},
			{3.75, 0, 1, 0.1},
			{4.09, 0, 1.33, 0.07},
		}, 2)

		sig = sc.Mix(sc.AR, tonesFrom(data, dur, fund)).Mul(sc.EnvGen{
			Env: sc.EnvPerc{
				Release: dur.Add(sc.C(-0.01)),
			},
			Done: sc.FreeEnclosing,
		}.Rate(sc.KR))
	)
	sig = sig.Mul(amp)

	return sc.Out{
		Bus:      sc.C(0),
		Channels: sc.Multi(sig, sig),
	}.Rate(sc.AR)
}

// tonesFrom creates harmonics from the provided data.
func tonesFrom(data [][4]float32, dur, fund sc.Input) []sc.Input {
	tones := make([]sc.Input, len(data))

	for i, row := range data {
		var (
			freq   = sc.C(row[0])
			offset = sc.C(row[1])
			amp    = sc.C(row[2])
			durr   = sc.C(row[3])

			ampenv = sc.EnvGen{
				Env: sc.EnvPerc{
					Release: dur.Mul(durr).Add(sc.C(-0.01)),
				},
			}.Rate(sc.KR)
		)
		tones[i] = sc.SinOsc{
			Freq: fund.Mul(freq).Add(offset),
		}.Rate(sc.AR).Mul(amp.Mul(ampenv))
	}
	return tones
}

// normalizeSum normalizes the specified column of data ala http://doc.sccode.org/Classes/Array.html#-normalizeSum
func normalizeSum(fs [][4]float32, i int) [][4]float32 {
	var sum float32
	for j := range fs {
		sum += fs[j][i]
	}
	for j := range fs {
		fs[j][i] /= sum
	}
	return fs
}
