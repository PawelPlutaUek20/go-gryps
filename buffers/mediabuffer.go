package buffers

import "slices"

type MediaData struct {
	SeqId    uint64
	Data     *[]byte
	Duration float64
}

type MediaBuffer struct {
	segments    []*MediaData
	duration    float64
	maxDuration float64
}

func NewMediaBuffer(maxDuration int) *MediaBuffer {
	return &MediaBuffer{
		segments:    make([]*MediaData, 0, maxDuration),
		maxDuration: float64(maxDuration),
	}
}

func (mb *MediaBuffer) Insert(segment *MediaData) {
	pos := 0
	for i := 0; i < len(mb.segments); i++ {
		seg := mb.segments[i]

		// Dont allow duplicates
		if segment.SeqId == seg.SeqId {
			return
		}

		if segment.SeqId < seg.SeqId {
			pos = i
			break
		}

		pos = i + 1
	}

	mb.segments = append(mb.segments, nil)
	copy(mb.segments[pos+1:], mb.segments[pos:])
	mb.segments[pos] = segment
	mb.duration += segment.Duration

	for mb.duration > mb.maxDuration && len(mb.segments) > 0 {
		mb.duration -= mb.segments[0].Duration
		copy(mb.segments, mb.segments[1:])
		mb.segments[len(mb.segments)-1] = nil
		mb.segments = mb.segments[:len(mb.segments)-1]
	}
}

func (mb *MediaBuffer) Contains(seqId uint64) bool {
	return slices.ContainsFunc(mb.segments, func(seg *MediaData) bool {
		return seg.SeqId == seqId
	})
}

func (mb *MediaBuffer) Segments() []*MediaData {
	return mb.segments
}
