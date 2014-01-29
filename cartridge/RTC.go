package cartridge

import (
	"time"
)

type RTC struct {
	Second  byte
	s       byte
	Minute  byte
	m       byte
	Hour    byte
	h       byte
	Day     byte
	d       byte
	Latched byte
	ticker  *time.Ticker
}

func NewRTC() *RTC {
	rtc := new(RTC)
	rtc.ticker = time.NewTicker(time.Second)

	go func() {
		for _ = range rtc.ticker.C {
			if rtc.s++; rtc.s >= 60 {
				rtc.s = 0
				rtc.m++
			}
			if rtc.m >= 60 {
				rtc.m = 0
				rtc.h++
			}
			if rtc.h >= 24 {
				rtc.h = 0
				rtc.d++
			}
		}
	}()

	return rtc
}

func (rtc *RTC) SetSecond(s byte) {
	if s >= 60 {
		s = 0
	}
	rtc.s = s
}

func (rtc *RTC) SetMinute(m byte) {
	if m >= 60 {
		m = 0
	}
	rtc.m = m
}

func (rtc *RTC) SetHour(h byte) {
	if h >= 24 {
		h = 0
	}
	rtc.h = h
}

func (rtc *RTC) SetDay(d byte) {
	rtc.d = d
}

func (rtc *RTC) Latch() {
	rtc.Second = rtc.s
	rtc.Minute = rtc.m
	rtc.Hour = rtc.h
	rtc.Day = rtc.d
	rtc.Latched = 1
}
