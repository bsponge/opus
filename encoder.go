// Copyright © 2015-2017 Go Opus Authors (see AUTHORS file)
//
// License for use of this code is detailed in the LICENSE file

package opus

import (
	"fmt"
	"unsafe"
)

/*
#cgo pkg-config: opus
#include <opus/opus.h>

void
bridge_encoder_set_dtx(OpusEncoder *st, opus_int32 use_dtx)
{
	opus_encoder_ctl(st, OPUS_SET_DTX(use_dtx));
}

opus_int32
bridge_encoder_get_dtx(OpusEncoder *st)
{
	opus_int32 dtx = 0;
	opus_encoder_ctl(st, OPUS_GET_DTX(&dtx));
	return dtx;
}

opus_int32
bridge_encoder_get_sample_rate(OpusEncoder *st)
{
	opus_int32 sample_rate = 0;
	opus_encoder_ctl(st, OPUS_GET_SAMPLE_RATE(&sample_rate));
	return sample_rate;
}


int
bridge_encoder_set_bitrate(OpusEncoder *st, opus_int32 bitrate)
{
	int res;
	res = opus_encoder_ctl(st, OPUS_SET_BITRATE(bitrate));
	return res;
}

int
bridge_encoder_get_bitrate(OpusEncoder *st, opus_int32 *bitrate)
{
	int res;
	res = opus_encoder_ctl(st, OPUS_GET_BITRATE(bitrate));
	return res;
}

int
bridge_encoder_set_complexity(OpusEncoder *st, opus_int32 complexity)
{
	int res;
	res = opus_encoder_ctl(st, OPUS_SET_COMPLEXITY(complexity));
	return res;
}

int
bridge_encoder_get_complexity(OpusEncoder *st, opus_int32 *complexity)
{
	int res;
	res = opus_encoder_ctl(st, OPUS_GET_COMPLEXITY(complexity));
	return res;
}

int
bridge_encoder_set_max_bandwidth(OpusEncoder *st, opus_int32 max_bw)
{
	int res;
	res = opus_encoder_ctl(st, OPUS_SET_MAX_BANDWIDTH(max_bw));
	return res;
}

int
bridge_encoder_get_max_bandwidth(OpusEncoder *st, opus_int32 *max_bw)
{
	int res;
	res = opus_encoder_ctl(st, OPUS_GET_MAX_BANDWIDTH(max_bw));
	return res;
}

// Access the preprocessor from CGO
const int CONST_BANDWIDTH_NARROWBAND = OPUS_BANDWIDTH_NARROWBAND;
const int CONST_BANDWIDTH_MEDIUMBAND = OPUS_BANDWIDTH_MEDIUMBAND;
const int CONST_BANDWIDTH_WIDEBAND = OPUS_BANDWIDTH_WIDEBAND;
const int CONST_BANDWIDTH_SUPERWIDEBAND = OPUS_BANDWIDTH_SUPERWIDEBAND;
const int CONST_BANDWIDTH_FULLBAND = OPUS_BANDWIDTH_FULLBAND;

const int CONST_BITRATE_AUTO = OPUS_AUTO;
const int CONST_BITRATE_MAX = OPUS_BITRATE_MAX;

*/
import "C"

type Bandwidth int

var (
	// 4 kHz passband
	Narrowband = Bandwidth(C.OPUS_BANDWIDTH_NARROWBAND)
	// 6 kHz passband
	Mediumband = Bandwidth(C.OPUS_BANDWIDTH_MEDIUMBAND)
	// 8 kHz passband
	Wideband = Bandwidth(C.OPUS_BANDWIDTH_WIDEBAND)
	// 12 kHz passband
	SuperWideband = Bandwidth(C.OPUS_BANDWIDTH_SUPERWIDEBAND)
	// 20 kHz passband
	Fullband = Bandwidth(C.OPUS_BANDWIDTH_FULLBAND)
)

var errEncUninitialized = fmt.Errorf("opus encoder uninitialized")

// Encoder contains the state of an Opus encoder for libopus.
type Encoder struct {
	p        *C.struct_OpusEncoder
	channels int
	// Memory for the encoder struct allocated on the Go heap to allow Go GC to
	// manage it (and obviate need to free())
	mem []byte
}

// NewEncoder allocates a new Opus encoder and initializes it with the
// appropriate parameters. All related memory is managed by the Go GC.
func NewEncoder(sample_rate int, channels int, application Application) (*Encoder, error) {
	var enc Encoder
	err := enc.Init(sample_rate, channels, application)
	if err != nil {
		return nil, err
	}
	return &enc, nil
}

// Init initializes a pre-allocated opus encoder. Unless the encoder has been
// created using NewEncoder, this method must be called exactly once in the
// life-time of this object, before calling any other methods.
func (enc *Encoder) Init(sample_rate int, channels int, application Application) error {
	if enc.p != nil {
		return fmt.Errorf("opus encoder already initialized")
	}
	if channels != 1 && channels != 2 {
		return fmt.Errorf("Number of channels must be 1 or 2: %d", channels)
	}
	size := C.opus_encoder_get_size(C.int(channels))
	enc.channels = channels
	enc.mem = make([]byte, size)
	enc.p = (*C.OpusEncoder)(unsafe.Pointer(&enc.mem[0]))
	errno := int(C.opus_encoder_init(
		enc.p,
		C.opus_int32(sample_rate),
		C.int(channels),
		C.int(application)))
	if errno != 0 {
		return Error(int(errno))
	}
	return nil
}

// Encode raw PCM data and store the result in the supplied buffer. On success,
// returns the number of bytes used up by the encoded data.
func (enc *Encoder) Encode(pcm []int16, data []byte) (int, error) {
	if enc.p == nil {
		return 0, errEncUninitialized
	}
	if len(pcm) == 0 {
		return 0, fmt.Errorf("opus: no data supplied")
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("opus: no target buffer")
	}
	// libopus talks about samples as 1 sample containing multiple channels. So
	// e.g. 20 samples of 2-channel data is actually 40 raw data points.
	if len(pcm)%enc.channels != 0 {
		return 0, fmt.Errorf("opus: input buffer length must be multiple of channels")
	}
	samples := len(pcm) / enc.channels
	n := int(C.opus_encode(
		enc.p,
		(*C.opus_int16)(&pcm[0]),
		C.int(samples),
		(*C.uchar)(&data[0]),
		C.opus_int32(cap(data))))
	if n < 0 {
		return 0, Error(n)
	}
	return n, nil
}

// Encode raw PCM data and store the result in the supplied buffer. On success,
// returns the number of bytes used up by the encoded data.
func (enc *Encoder) EncodeFloat32(pcm []float32, data []byte) (int, error) {
	if enc.p == nil {
		return 0, errEncUninitialized
	}
	if len(pcm) == 0 {
		return 0, fmt.Errorf("opus: no data supplied")
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("opus: no target buffer")
	}
	if len(pcm)%enc.channels != 0 {
		return 0, fmt.Errorf("opus: input buffer length must be multiple of channels")
	}
	samples := len(pcm) / enc.channels
	n := int(C.opus_encode_float(
		enc.p,
		(*C.float)(&pcm[0]),
		C.int(samples),
		(*C.uchar)(&data[0]),
		C.opus_int32(cap(data))))
	if n < 0 {
		return 0, Error(n)
	}
	return n, nil
}

// UseDTX configures the encoder's use of discontinuous transmission (DTX).
func (enc *Encoder) UseDTX(use bool) {
	dtx := 0
	if use {
		dtx = 1
	}
	C.bridge_encoder_set_dtx(enc.p, C.opus_int32(dtx))
}

// DTX reports whether this encoder is configured to use discontinuous
// transmission (DTX).
func (enc *Encoder) DTX() bool {
	dtx := C.bridge_encoder_get_dtx(enc.p)
	return dtx != 0
}

// SampleRate returns the encoder sample rate in Hz.
func (enc *Encoder) SampleRate() int {
	return int(C.bridge_encoder_get_sample_rate(enc.p))
}

// SetBitrate sets the bitrate of the Encoder
func (enc *Encoder) SetBitrate(bitrate int) error {
	res := C.bridge_encoder_set_bitrate(enc.p, C.opus_int32(bitrate))
	if res != C.OPUS_OK {
		return Error(res)
	}
	return nil
}

// SetBitrateAuto will allow the encoder to automatically set the bitrate
func (enc *Encoder) SetBitrateAuto() error {
	res := C.bridge_encoder_set_bitrate(enc.p, C.opus_int32(C.CONST_BITRATE_AUTO))
	if res != C.OPUS_OK {
		return Error(res)
	}
	return nil
}

// SetBitrateMax causes the encoder to use as much rate as it can. This can be
// useful for controlling the rate by adjusting the output buffer size.
func (enc *Encoder) SetBitrateMax() error {
	res := C.bridge_encoder_set_bitrate(enc.p, C.opus_int32(C.CONST_BITRATE_MAX))
	if res != C.OPUS_OK {
		return Error(res)
	}
	return nil
}

// Bitrate returns the bitrate of the Encoder
func (enc *Encoder) Bitrate() (int, error) {
	var bitrate C.opus_int32
	res := C.bridge_encoder_get_bitrate(enc.p, &bitrate)
	if res != C.OPUS_OK {
		return 0, Error(res)
	}
	return int(bitrate), nil
}

// SetComplexity sets the encoder's computational complexity
func (enc *Encoder) SetComplexity(complexity int) error {
	res := C.bridge_encoder_set_complexity(enc.p, C.opus_int32(complexity))
	if res != C.OPUS_OK {
		return Error(res)
	}
	return nil
}

// Complexity returns the computational complexity used by the encoder
func (enc *Encoder) Complexity() (int, error) {
	var complexity C.opus_int32
	res := C.bridge_encoder_get_complexity(enc.p, &complexity)
	if res != C.OPUS_OK {
		return 0, Error(res)
	}
	return int(complexity), nil
}

// SetMaxBandwidth configures the maximum bandpass that the encoder will select
// automatically
func (enc *Encoder) SetMaxBandwidth(maxBw Bandwidth) error {
	res := C.bridge_encoder_set_max_bandwidth(enc.p, C.opus_int32(maxBw))
	if res != C.OPUS_OK {
		return Error(res)
	}
	return nil
}

// MaxBandwidth gets the encoder's configured maximum allowed bandpass.
func (enc *Encoder) MaxBandwidth() (Bandwidth, error) {
	var maxBw C.opus_int32
	res := C.bridge_encoder_get_max_bandwidth(enc.p, &maxBw)
	if res != C.OPUS_OK {
		return 0, Error(res)
	}
	return Bandwidth(maxBw), nil
}
